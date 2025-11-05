package models

import (
	"context"
	"fmt"
	"math"
	"time"

	log "github.com/gophish/gophish/logger"
	"github.com/gophish/gophish/smser"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/pinpointsmsvoicev2"
	"github.com/aws/aws-sdk-go-v2/service/pinpointsmsvoicev2/types"
)

// SmsLog is a struct that holds information about an sms that is to be
// sent out.
type SmsLog struct {
	Id          int64     `json:"-" gorm:"column:id; primary_key:yes"`
	UserId      int64     `json:"-" gorm:"column:user_id"`
	CampaignId  int64     `json:"campaign_id" gorm:"column:campaign_id"`
	RId         string    `json:"id" gorm:"column:r_id"`
	SendDate    time.Time `json:"send_date" gorm:"column:send_date"`
	SendAttempt int       `json:"send_attempt" gorm:"column:send_attempt"`
	Processing  bool      `json:"-" gorm:"column:processing"`
	Target      string    `json:"target" gorm:"column:target"`

	cachedCampaign *Campaign
}

// TableName specifies the database tablename for Gorm to use
func (s SmsLog) TableName() string {
	return "sms_logs"
}

// GenerateSmsLog creates a new smslog for the given campaign and
// result. It sets the initial send date to match the campaign's launch date.
func GenerateSmsLog(c *Campaign, r *Result, sendDate time.Time) error {
	s := &SmsLog{
		UserId:     c.UserId,
		CampaignId: c.Id,
		RId:        r.RId,
		SendDate:   sendDate,
	}
	return db.Save(s).Error
}

// Backoff sets the SmsLog SendDate to be the next entry in an exponential
// backoff. ErrMaxRetriesExceeded is thrown if this smslog has been retried
// too many times. Backoff also unlocks the smslog so that it can be processed
// again in the future.
func (s *SmsLog) Backoff(reason error) error {
	r, err := GetResult(s.RId)
	if err != nil {
		return err
	}
	if s.SendAttempt == MaxSendAttempts {
		r.HandleEmailError(ErrMaxSendAttempts)
		return ErrMaxSendAttempts
	}
	// Add an error, since we had to backoff because of a
	// temporary error of some sort during the SMS transaction
	s.SendAttempt++
	backoffDuration := math.Pow(2, float64(s.SendAttempt))
	s.SendDate = s.SendDate.Add(time.Minute * time.Duration(backoffDuration))
	err = db.Save(s).Error
	if err != nil {
		return err
	}
	err = r.HandleEmailBackoff(reason, s.SendDate)
	if err != nil {
		return err
	}
	err = s.Unlock()
	return err
}

// Unlock removes the processing flag so the smslog can be processed again
func (s *SmsLog) Unlock() error {
	s.Processing = false
	return db.Save(&s).Error
}

// Lock sets the processing flag so that other processes cannot modify the smslog
func (s *SmsLog) Lock() error {
	s.Processing = true
	return db.Save(&s).Error
}

// Error sets the error status on the models.Result that the
// smslog refers to. Since SmsLog errors are permanent,
// this action also deletes the smslog.
func (s *SmsLog) Error(e error) error {
	r, err := GetResult(s.RId)
	if err != nil {
		log.Warn(err)
		return err
	}
	err = r.HandleEmailError(e)
	if err != nil {
		log.Warn(err)
		return err
	}
	err = db.Delete(s).Error
	return err
}

// Success deletes the smslog from the database and updates the underlying
// campaign result.
func (s *SmsLog) Success() error {
	r, err := GetResult(s.RId)
	if err != nil {
		return err
	}
	err = r.HandleSMSSent()
	if err != nil {
		return err
	}
	err = db.Delete(s).Error
	return err
}

// CacheCampaign allows bulk-mail workers to cache the otherwise expensive
// campaign lookup operation by providing a pointer to the campaign here.
func (s *SmsLog) CacheCampaign(campaign *Campaign) error {
	if campaign.Id != s.CampaignId {
		return fmt.Errorf("incorrect campaign provided for caching. expected %d got %d", s.CampaignId, campaign.Id)
	}
	s.cachedCampaign = campaign
	return nil
}

// Generate fills in the details of a smser.SNSMessage instance with
// information from the campaign and recipient listed in the smslog.
func (s *SmsLog) Generate(msg *smser.SNSMessage) error {
	r, err := GetResult(s.RId)
	if err != nil {
		return err
	}

	c := s.cachedCampaign
	if c == nil {
		campaign, err := GetCampaignSMSContext(s.CampaignId, s.UserId)
		if err != nil {
			return err
		}
		c = &campaign
	}

	ptx, err := NewPhishingTemplateContextSms(c, r.BaseRecipient, r.RId)
	if err != nil {
		return err
	}

	if c.Template.Text != "" {
		text, err := ExecuteTemplate(c.Template.Text, ptx)
		if err != nil {
			log.Warn(err)
		}

    	// Create AWS SMSVoiceV2 client
		cfg, err := config.LoadDefaultConfig(context.Background(),
			config.WithRegion(c.SMS.AWSRegion),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(c.SMS.AWSAccessKeyId, c.SMS.AWSSecretKey, "")),
		)
		if err != nil {
			return fmt.Errorf("failed to load AWS config: %v", err)
		}

		msg.Client = pinpointsmsvoicev2.NewFromConfig(cfg)
		
		// Format phone number (E.164 format)
		phoneNumber := s.Target
		if len(phoneNumber) == 10 && phoneNumber[0] != '+' {
			// Assume US number if 10 digits without country code
			phoneNumber = "+1" + phoneNumber
		} else if len(phoneNumber) == 11 && phoneNumber[0] == '1' {
			// Add + if missing for US numbers
			phoneNumber = "+" + phoneNumber
		}
		
		log.Infof("Sending SMS to phone number: %s (formatted from: %s)", phoneNumber, s.Target)
		
		msg.Params = pinpointsmsvoicev2.SendTextMessageInput{
			MessageBody:            aws.String(text),
			DestinationPhoneNumber: aws.String(phoneNumber),
			OriginationPhoneNumber: aws.String(c.SMS.SMSFrom),
			MessageType:            types.MessageTypePromotional,
		}
	} else {
		return fmt.Errorf("No text template specified")
	}

	return nil
}

// GetQueuedSmsLogs returns the sms logs that are queued up for the given minute.
func GetQueuedSmsLogs(t time.Time) ([]*SmsLog, error) {
	sms := []*SmsLog{}
	err := db.Where("send_date <= ? AND processing = ?", t, false).
		Find(&sms).Error
	if err != nil {
		log.Warn(err)
	}
	return sms, err
}

// GetSmsLogsByCampaign returns all of the sms logs for a given campaign.
func GetSmsLogsByCampaign(cid int64) ([]*SmsLog, error) {
	sms := []*SmsLog{}
	err := db.Where("campaign_id = ?", cid).Find(&sms).Error
	return sms, err
}

// LockSmsLogs locks or unlocks a slice of smslogs for processing.
func LockSmsLogs(sms []*SmsLog, lock bool) error {
	tx := db.Begin()
	for i := range sms {
		sms[i].Processing = lock
		err := tx.Save(sms[i]).Error
		if err != nil {
			tx.Rollback()
			return err
		}
	}
	tx.Commit()
	return nil
}

// UnlockAllSmsLogs removes the processing lock for all smslogs
// in the database. This is intended to be called when Gophish is started
// so that any previously locked smslogs can resume processing.
func UnlockAllSmsLogs() error {
	return db.Model(&SmsLog{}).Update("processing", false).Error
}
