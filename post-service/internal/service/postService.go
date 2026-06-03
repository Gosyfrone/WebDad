package service

import (
	"context"
	"time"

	"github.com/webdad/post-service/internal/models"
	"github.com/webdad/post-service/internal/repository"
)

type PostService struct {
	repo *repository.PostRepository
}

func NewPostService(r *repository.PostRepository) *PostService {
	return &PostService{repo: r}
}

func (s *PostService) CreatePost(ctx context.Context, authorID, content string) error {
	post := &models.Post{
		AuthorID:  authorID,
		Content:   content,
		CreatedAt: time.Now(),
	}

	return s.repo.Create(ctx, post)
}

func (s *PostService) GetPosts(ctx context.Context) ([]models.Post, error) {
    return s.repo.GetAll(ctx)
}

func (s *PostService) GetPost(ctx context.Context, id string) (*models.Post, error) {
    return s.repo.Get(ctx,id)
}

func (s *PostService) DeletePost(ctx context.Context, id string) (error) {
    return s.repo.Delete(ctx,id)
}

func (s *PostService) UpdatePost(ctx context.Context, id string, content string) (*models.Post, error) {
    return s.repo.Update(ctx, id, content)
}