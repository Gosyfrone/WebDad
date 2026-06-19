// Package storage encapsule le stockage objet (MinIO, S3-compatible). C'est la
// « base de données » du media-service : il y range des octets opaques sous un
// id aléatoire, avec l'identifiant du propriétaire en métadonnée objet. Le
// service ne tient AUCUNE base relationnelle/document à côté.
package storage

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// ownerMetaKey : clé de métadonnée objet portant l'id du propriétaire.
// MinIO la préfixe "X-Amz-Meta-" et canonicalise la casse à la lecture.
const ownerMetaKey = "owner-id"

// ErrNotFound : objet absent (→ 404 côté handler).
var ErrNotFound = errors.New("média introuvable")

// Store : client MinIO + bucket cible.
type Store struct {
	client *minio.Client
	bucket string
}

// New connecte MinIO et garantit l'existence du bucket (idempotent) → le
// service est autonome, sans provisioning externe (même esprit que les
// EnsureSchema des services à base de données).
func New(endpoint, accessKey, secretKey string, useSSL bool, bucket string) (*Store, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	exists, err := client.BucketExists(ctx, bucket)
	if err != nil {
		return nil, err
	}
	if !exists {
		if err := client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, err
		}
	}

	return &Store{client: client, bucket: bucket}, nil
}

// Put range les octets sous `id`, avec le Content-Type réel et le propriétaire
// en métadonnée. `size` doit être la taille exacte du flux.
func (s *Store) Put(ctx context.Context, id string, r io.Reader, size int64, contentType, ownerID string) error {
	_, err := s.client.PutObject(ctx, s.bucket, id, r, size, minio.PutObjectOptions{
		ContentType:  contentType,
		UserMetadata: map[string]string{ownerMetaKey: ownerID},
	})
	return err
}

// PutVariant range une variante dérivée d'un média. Les variantes vivent sous
// un préfixe lié à l'id original afin de pouvoir les purger ensemble.
func (s *Store) PutVariant(ctx context.Context, originalID, variant string, r io.Reader, size int64, contentType, ownerID string) error {
	return s.Put(ctx, VariantKey(originalID, variant), r, size, contentType, ownerID)
}

// Open ouvre l'objet en lecture et renvoie son ObjectInfo. L'appel à Stat()
// matérialise un éventuel 404 (objet absent) → ErrNotFound. L'objet retourné
// est un io.ReadSeekCloser (compatible http.ServeContent pour les Range).
func (s *Store) Open(ctx context.Context, id string) (*minio.Object, minio.ObjectInfo, error) {
	obj, err := s.client.GetObject(ctx, s.bucket, id, minio.GetObjectOptions{})
	if err != nil {
		return nil, minio.ObjectInfo{}, err
	}
	info, err := obj.Stat()
	if err != nil {
		_ = obj.Close()
		if isNotFound(err) {
			return nil, minio.ObjectInfo{}, ErrNotFound
		}
		return nil, minio.ObjectInfo{}, err
	}
	return obj, info, nil
}

// Stat renvoie les métadonnées sans ouvrir de flux (utilisé pour DELETE).
func (s *Store) Stat(ctx context.Context, id string) (minio.ObjectInfo, error) {
	info, err := s.client.StatObject(ctx, s.bucket, id, minio.StatObjectOptions{})
	if err != nil {
		if isNotFound(err) {
			return minio.ObjectInfo{}, ErrNotFound
		}
		return minio.ObjectInfo{}, err
	}
	return info, nil
}

// Remove supprime l'objet.
func (s *Store) Remove(ctx context.Context, id string) error {
	return s.client.RemoveObject(ctx, s.bucket, id, minio.RemoveObjectOptions{})
}

// RemoveMediaSet supprime l'original et toutes ses variantes connues/présentes.
func (s *Store) RemoveMediaSet(ctx context.Context, id string) error {
	if err := s.Remove(ctx, id); err != nil {
		return err
	}
	prefix := id + "/"
	for obj := range s.client.ListObjects(ctx, s.bucket, minio.ListObjectsOptions{Prefix: prefix, Recursive: true}) {
		if obj.Err != nil {
			return obj.Err
		}
		if err := s.Remove(ctx, obj.Key); err != nil {
			return err
		}
	}
	return nil
}

// RemoveByOwner supprime TOUS les objets appartenant à `ownerID` (effacement
// RGPD). Parcourt le bucket (scan O(n)) et lit la métadonnée propriétaire via
// Stat (fiable, contrairement à la métadonnée parfois absente du listing).
// Renvoie le nombre d'objets supprimés. Pas de base à côté → un scan est le
// prix de l'autonomie du service (perspective : index propriétaire→objets).
func (s *Store) RemoveByOwner(ctx context.Context, ownerID string) (int, error) {
	count := 0
	for obj := range s.client.ListObjects(ctx, s.bucket, minio.ListObjectsOptions{Recursive: true}) {
		if obj.Err != nil {
			return count, obj.Err
		}
		info, err := s.Stat(ctx, obj.Key)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				continue // course : objet déjà supprimé entre-temps
			}
			return count, err
		}
		if OwnerOf(info) != ownerID {
			continue
		}
		if err := s.Remove(ctx, obj.Key); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

// OwnerOf extrait l'id du propriétaire depuis les métadonnées d'un ObjectInfo.
func OwnerOf(info minio.ObjectInfo) string {
	return info.Metadata.Get("X-Amz-Meta-" + ownerMetaKey)
}

// VariantKey construit la clé objet d'une variante. Le nom de variante est
// borné à un segment simple pour éviter toute traversée de préfixe.
func VariantKey(originalID, variant string) string {
	return originalID + "/" + strings.Trim(variant, "/")
}

// isNotFound reconnaît l'erreur « clé inexistante » de MinIO.
func isNotFound(err error) bool {
	resp := minio.ToErrorResponse(err)
	return resp.Code == "NoSuchKey" || resp.StatusCode == http.StatusNotFound
}
