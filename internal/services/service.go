package services

import (
	"encoding/json"
	"fmt"
	"net/http"

	local_repo "github.com/avantifellows/nex-gen-cms/internal/repositories/local"
	remote_repo "github.com/avantifellows/nex-gen-cms/internal/repositories/remote"
	"github.com/thoas/go-funk"
)

type Service[T any] struct {
	cacheRepository *local_repo.CacheRepository
	apiRepository   *remote_repo.APIRepository
}

// NewService creates a new instance of Service
func NewService[T any](cacheRepo *local_repo.CacheRepository, apiRepo *remote_repo.APIRepository) *Service[T] {
	return &Service[T]{
		cacheRepository: cacheRepo,
		apiRepository:   apiRepo,
	}
}

// GetList returns data from cache or API
func (s *Service[T]) GetList(urlEndPoint string, cacheKey string, onlyCache bool) (*[]*T, error) {

	// Check if data is in cache
	if list, found := s.cacheRepository.Get(cacheKey); found {
		return list.(*[]*T), nil
	}

	if onlyCache {
		return nil, nil
	}

	// Otherwise, fetch from API
	respBytes, err := s.apiRepository.CallAPI(urlEndPoint, http.MethodGet, nil)
	if err != nil {
		return nil, err
	}

	// Unmarshal the response bytes into pointer to list
	var list []*T
	if err := json.Unmarshal(respBytes, &list); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	// Cache the data
	s.cacheRepository.Set(cacheKey, &list)

	return &list, nil
}

func (s *Service[T]) GetObject(objIdStr string, objFindingPredicate func(*T) bool, cacheKey string,
	urlEndPoint string) (*T, error) {

	// check in the cache for list
	list, _ := s.GetList(urlEndPoint, cacheKey, true)
	objPtr := new(T)
	if list != nil {
		objPtr = funk.Find(*list, objFindingPredicate).(*T)

	} else {
		// call api to fetch single object
		respBytes, err := s.apiRepository.CallAPI(urlEndPoint+"/"+objIdStr, http.MethodGet, nil)
		if err != nil {
			return nil, err
		}

		// Unmarshal the response bytes into pointer to object
		if err := json.Unmarshal(respBytes, objPtr); err != nil {
			return nil, fmt.Errorf("error parsing response: %v", err)
		}
	}
	return objPtr, nil
}

func (s *Service[T]) UpdateObject(objIdStr string, urlEndPoint string, body any) (*T, error) {

	respBytes, err := s.apiRepository.CallAPI(urlEndPoint+"/"+objIdStr, http.MethodPatch, body)
	if err != nil {
		return nil, err
	}

	objPtr := new(T)
	// Unmarshal the response bytes into pointer to object
	if err := json.Unmarshal(respBytes, objPtr); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	return objPtr, nil
}

func (s *Service[T]) AddObject(body any, cacheKey string, urlEndPoint string) (*T, error) {
	// add in remote db
	respBytes, err := s.apiRepository.CallAPI(urlEndPoint, http.MethodPost, body)
	if err != nil {
		return nil, err
	}

	objPtr := new(T)
	// Unmarshal the response bytes into pointer to object
	if err := json.Unmarshal(respBytes, objPtr); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	// Add in cache
	list, _ := s.GetList(urlEndPoint, cacheKey, true)
	if list != nil {
		*list = append(*list, objPtr)
	}
	return objPtr, nil
}

func (s *Service[T]) DeleteObject(objIdStr string, objKeepingPredicate func(*T) bool, cacheKey string,
	urlEndPoint string) error {
	_, err := s.apiRepository.CallAPI(urlEndPoint+"/"+objIdStr, http.MethodDelete, nil)
	if err != nil {
		return err
	}

	// as deleted from api without any error, now delete from cache also
	list, _ := s.GetList(urlEndPoint, cacheKey, true)
	if list != nil {
		*list = funk.Filter(*list, objKeepingPredicate).([]*T)
	}
	return nil
}
