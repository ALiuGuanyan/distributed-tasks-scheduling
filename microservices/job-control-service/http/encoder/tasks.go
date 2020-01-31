package encoder

import (
	"context"
	"encoding/json"
	"github.com/ALiuGuanyan/distributed-task-scheduling/microservices/job-control-service/responses"
	"github.com/ALiuGuanyan/distributed-task-scheduling/microservices/job-control-service/utils/appErrors"
	"net/http"
)

func(ec *ImplEncoder) EncodeSaveOneTaskResponse(_ context.Context, w http.ResponseWriter, response interface{}) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	res, ok := response.(responses.Response)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		return appErrors.AssertError
	}

	str, err := json.Marshal(res)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return err
	}

	w.WriteHeader(http.StatusCreated)
	w.Write(str)
	return nil
}

func (ec *ImplEncoder) EncodeDeleteOneTaskResponse(_ context.Context, w http.ResponseWriter, response interface{}) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	res, ok := response.(responses.Response)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		return appErrors.AssertError
	}

	str, err := json.Marshal(res)
	if err != nil {
		return err
	}
	w.WriteHeader(http.StatusOK)
	w.Write(str)
	return nil
}

func (ec *ImplEncoder) EncodeGetOneTaskResponse(_ context.Context, w http.ResponseWriter, response interface{}) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	res, ok := response.(responses.GetOneTaskResponse)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		return appErrors.AssertError
	}

	str, err := json.Marshal(res)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return err
	}
	w.WriteHeader(http.StatusFound)
	w.Write(str)
	return nil
}

func (ec *ImplEncoder) EncodeGetAllTasksResponse(_ context.Context, w http.ResponseWriter, response interface{}) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	res, ok := response.(responses.GetAllTasksResponse)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		return appErrors.AssertError
	}



	str, err := json.Marshal(res)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return err
	}

	w.WriteHeader(http.StatusFound)
	w.Write(str)
	return nil
}
