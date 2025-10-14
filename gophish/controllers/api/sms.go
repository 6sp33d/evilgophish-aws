package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	ctx "github.com/gophish/gophish/context"
	log "github.com/gophish/gophish/logger"
	"github.com/gophish/gophish/models"
	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
)

// SMSProfiles handles requests for the /api/sms/ endpoint
func (as *Server) SMSProfiles(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == "GET":
		ss, err := models.GetSMSs(ctx.Get(r, "user_id").(int64))
		if err != nil {
			log.Error(err)
		}
		JSONResponse(w, ss, http.StatusOK)
	//POST: Create a new SMS and return it as JSON
	case r.Method == "POST":
		s := models.SMS{}
		// Put the request into a page
		err := json.NewDecoder(r.Body).Decode(&s)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Invalid request"}, http.StatusBadRequest)
			return
		}
		// Check to make sure the name is unique
		_, err = models.GetSMSByName(s.Name, ctx.Get(r, "user_id").(int64))
		if err != gorm.ErrRecordNotFound {
			JSONResponse(w, models.Response{Success: false, Message: "SMS name already in use"}, http.StatusConflict)
			log.Error(err)
			return
		}
		s.ModifiedDate = time.Now().UTC()
		s.UserId = ctx.Get(r, "user_id").(int64)
		err = models.PostSMS(&s)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, s, http.StatusCreated)
	}
}

// SendingProfile contains functions to handle the GET'ing, DELETE'ing, and PUT'ing
// of a SMTP object
func (as *Server) SMSProfile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 0, 64)
	s, err := models.GetSMS(id, ctx.Get(r, "user_id").(int64))
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "SMS not found"}, http.StatusNotFound)
		return
	}
	switch {
	case r.Method == "GET":
		JSONResponse(w, s, http.StatusOK)
	case r.Method == "DELETE":
		err = models.DeleteSMS(id, ctx.Get(r, "user_id").(int64))
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Error deleting SMS"}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, models.Response{Success: true, Message: "SMS Deleted Successfully"}, http.StatusOK)
	case r.Method == "PUT":
		s := models.SMS{}
		err = json.NewDecoder(r.Body).Decode(&s)
		if err != nil {
			log.Error(err)
		}
		if s.Id != id {
			JSONResponse(w, models.Response{Success: false, Message: "/:id and /:sms_id mismatch"}, http.StatusBadRequest)
			return
		}
		err = s.Validate()
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadRequest)
			return
		}
		s.ModifiedDate = time.Now().UTC()
		s.UserId = ctx.Get(r, "user_id").(int64)
		err = models.PutSMS(&s)
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: "Error updating page"}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, s, http.StatusOK)
	}
}

// PhoneNumberRequest represents the request to fetch phone numbers
type PhoneNumberRequest struct {
	AccessKeyID string `json:"access_key_id"`
	SecretKey   string `json:"secret_key"`
	Region      string `json:"region"`
}

// PhoneNumber represents a phone number from AWS
type PhoneNumber struct {
	Number string `json:"number"`
	Status string `json:"status"`
}

// SMSPhoneNumbers handles requests for fetching phone numbers from AWS
func (as *Server) SMSPhoneNumbers(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		JSONResponse(w, models.Response{Success: false, Message: "Method not allowed"}, http.StatusMethodNotAllowed)
		return
	}

	var req PhoneNumberRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Invalid request"}, http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.AccessKeyID == "" || req.SecretKey == "" || req.Region == "" {
		JSONResponse(w, models.Response{Success: false, Message: "Missing required fields"}, http.StatusBadRequest)
		return
	}

	// TODO: Implement actual AWS API integration
	// For now, we'll return a mock response
	// In a real implementation, you would:
	// 1. Create AWS config with provided credentials
	// 2. Use AWS Pinpoint or SNS service to list phone numbers
	// 3. Parse the response and extract phone numbers
	
	log.Infof("Fetching phone numbers for region: %s", req.Region)
	
	// Mock phone numbers - replace with actual AWS API call
	phoneNumbers := []PhoneNumber{
		{Number: "+1234567890", Status: "ACTIVE"},
		{Number: "+1987654321", Status: "ACTIVE"},
		{Number: "+1555123456", Status: "ACTIVE"},
	}

	JSONResponse(w, phoneNumbers, http.StatusOK)
}