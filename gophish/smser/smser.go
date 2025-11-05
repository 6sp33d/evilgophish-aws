package smser

import (
	"context"

	log "github.com/gophish/gophish/logger"
    "github.com/aws/aws-sdk-go-v2/service/smsvoicev2"
)

type SNSMessage struct {
    Client *smsvoicev2.Client
    Params smsvoicev2.SendTextMessageInput
}

// Smser is an interface that defines an object used to queue and
// send mailer.Sms instances.
type Smser interface {
	Start(ctx context.Context)
	Queue([]Sms)
}

// Sms is an interface that handles the common operations for sms messages
type Sms interface {
	Error(err error) error
	Success() error
	Generate(msg *SNSMessage) error
	Backoff(err error) error
}

// SmsWorker is the worker that receives slices of sms's
type SmsWorker struct {
	queue chan []Sms
}

// NewSmsWorker returns an instance of SmsWorker with the mail queue
// initialized.
func NewSmsWorker() *SmsWorker {
	return &SmsWorker{
		queue: make(chan []Sms),
	}
}

// Start launches the mail worker to begin listening on the Queue channel
// for new slices of Sms instances to process.
func (sw *SmsWorker) Start(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case sms := <-sw.queue:
			go func(ctx context.Context, sms []Sms) {
				sendSms(ctx, sms)
			}(ctx, sms)
		}
	}
}

// Queue sends the provided mail to the internal queue for processing.
func (sw *SmsWorker) Queue(sms []Sms) {
	sw.queue <- sms
}

// sendSms attempts to send the provided Sms instances.
// If the context is cancelled before all of the sms are sent,
// sendSms just returns and does not modify those sms's.
func sendSms(ctx context.Context, sms []Sms) {
	log.Infof("sendSms called with %d SMS messages", len(sms))
	for _, s := range sms {
		select {
		case <-ctx.Done():
			log.Info("Context cancelled, stopping SMS sending")
			return
		default:
			break
		}
		// Generate the message
		log.Info("Generating SNS message")
		message := &SNSMessage{}
		err := s.Generate(message)
		if err != nil {
			log.Warnf("Error generating message: %v", err)
			s.Error(err)
			continue
		}
        log.Infof("Message generated successfully, about to send via SMSVoiceV2")
        // Send the message
        result, err := message.Client.SendTextMessage(ctx, &message.Params)
		if err != nil {
            log.Warnf("Error sending SMS via SMSVoiceV2: %v", err)
			s.Backoff(err)
			continue
		}
        log.Infof("SMS sent successfully via AWS SMSVoiceV2, MessageId: %v", result.MessageId)
		s.Success()
	}
	log.Info("sendSms completed")
}
