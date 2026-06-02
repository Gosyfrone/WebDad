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

func (s *PostService) CreatePost(ctx context.Context, published_by, content string) error {
    post := &models.Post{
        PublishedBy:   published_by,
        Content: content,
        PublishedAt: time.Now(),
    }

    return s.repo.Create(ctx, post)
}

func (s *PostService) GetPosts(ctx context.Context) ([]models.Post, error) {
    return s.repo.GetAll(ctx)
}