package handlerutils

import (
	"github.com/avantifellows/nex-gen-cms/internal/models"
	"github.com/avantifellows/nex-gen-cms/internal/services"
	"github.com/avantifellows/nex-gen-cms/utils"
)

const ChaptersEndPoint = "chapter"
const ChaptersKey = "chapters"

func GetChapterByID(chapterIDStr string, chaptersService *services.Service[models.Chapter]) (*models.Chapter, int, error) {
	return GetEntityByID(
		chapterIDStr,
		chaptersService,
		ChaptersKey,
		ChaptersEndPoint,
		utils.StringToIntType[int16],
		func(c *models.Chapter, id int16) bool { return c.ID == id },
		"Chapter",
	)
}
