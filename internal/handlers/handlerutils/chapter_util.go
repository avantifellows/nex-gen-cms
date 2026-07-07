package handlerutils

import (
	"fmt"
	"net/http"

	"github.com/avantifellows/nex-gen-cms/internal/models"
	"github.com/avantifellows/nex-gen-cms/internal/services"
	"github.com/avantifellows/nex-gen-cms/utils"
)

const ChaptersEndPoint = "chapter"
const ChaptersKey = "chapters"

func GetChapterById(chapterIdStr string, chaptersService *services.Service[models.Chapter]) (*models.Chapter, int, error) {
	chapterId, err := utils.StringToIntType[int16](chapterIdStr)
	if err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("invalid Chapter ID: %w", err)
	}

	selectedChapterPtr, err := chaptersService.GetObject(chapterIdStr,
		func(chapter *models.Chapter) bool {
			return (*chapter).ID == chapterId
		}, ChaptersKey, ChaptersEndPoint)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("error fetching chapter: %v", err)
	}

	return selectedChapterPtr, http.StatusOK, nil
}
