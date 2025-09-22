package handlers

import (
	"net/http"

	"github.com/avantifellows/nex-gen-cms/internal/views"
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
	views.ExecuteTemplates(responseWriter, nil, nil, baseTemplate, booksTemplate)
}
