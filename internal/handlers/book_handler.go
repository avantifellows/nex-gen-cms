package handlers

import (
	"net/http"

	local_repo "github.com/avantifellows/nex-gen-cms/internal/repositories/local"
)

const booksTemplate = "books.html"

type BooksHandler struct {
	// service *services.Service[models.Test]
}

func NewBooksHandler( /*service *services.Service[models.Test]*/ ) *BooksHandler {
	return &BooksHandler{
		// service: service,
	}
}

func (h *BooksHandler) LoadBooks(responseWriter http.ResponseWriter, request *http.Request) {
	local_repo.ExecuteTemplates(responseWriter, nil, nil, baseTemplate, booksTemplate)
}
