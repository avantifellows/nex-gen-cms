package services

import (
	"github.com/avantifellows/nex-gen-cms/internal/models"
	local_repo "github.com/avantifellows/nex-gen-cms/internal/repositories/local"
	remote_repo "github.com/avantifellows/nex-gen-cms/internal/repositories/remote"
	"github.com/thoas/go-funk"
)

type ChapterService struct {
	Service[models.Chapter]
}

// NewService creates a new instance of Service
func NewChapterService(cacheRepo *local_repo.CacheRepository, apiRepo *remote_repo.APIRepository) *ChapterService {
	return &ChapterService{
		Service[models.Chapter]{
			cacheRepository: cacheRepo,
			apiRepository:   apiRepo,
		},
	}
}

func (s *ChapterService) UpdateChapter(chapterIdStr string, chapterCode string, chapterName string, chaptersKey string,
	chapterFindingPredicate func(*models.Chapter) bool, chaptersEndPoint string) (*models.Chapter, error) {
	// Update in cache
	list, _ := s.GetList(chaptersEndPoint, chaptersKey, true)
	if list != nil {
		selectedChapterPtr := funk.Find(*list, chapterFindingPredicate).(*models.Chapter)
		if selectedChapterPtr != nil {
			selectedChapterPtr.UpdateProperties(chapterCode, chapterName)
		}
	}

	// Update on server
	/**
	  this chapter pointer is created just to call BuildMap() method. If we create function instead of
	  this method, then we won't be able to create functions with same name in other models, hence it is
	  created as method
	*/
	dummyChapterPtr := &models.Chapter{}
	chapterMap := dummyChapterPtr.BuildMap(chapterCode, chapterName)

	// call api to update chapter
	resultChapterPtr, err := s.Service.UpdateObject(chapterIdStr, chaptersEndPoint, chapterMap)
	if err != nil {
		return nil, err
	}
	return resultChapterPtr, nil
}
