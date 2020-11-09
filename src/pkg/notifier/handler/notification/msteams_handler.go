package notification

import (
	"errors"
	"fmt"

	"bytes"
	"encoding/json"
	"github.com/goharbor/harbor/src/common/job/models"
	"github.com/goharbor/harbor/src/jobservice/job"
	"github.com/goharbor/harbor/src/pkg/notification"
	"github.com/goharbor/harbor/src/pkg/notifier/model"
	"strings"
	"text/template"
)

const (
	// MSTeamsBodyTemplate defines MSTeams request body template
	MSTeamsBodyTemplate = `{
    "@type": "MessageCard",
    "@context": "https://schema.org/extensions",
    "summary": "Harbor Notification",
    "themeColor": "0078D7",
    "title": "{{.Type}}",
    "sections": [
        {
            "activityTitle": "registry.dev.finworks.com/cbpm/cbpm-app",
            "activitySubtitle": "<!date^{{.OccurAt}}^{date} at {time}|February 18th, 2014 at 6:39 AM PST>",           
            "activityImage": "https://branding.cncf.io/img/projects/harbor/icon/color/harbor-icon-color.png",
            "startGroup": true,
            "facts": [ 
                {
                    "name": "Scan Status:",
                    "value": "Success"
                },
                {
                    "name": "Severity:",
                    "value": "Low"
                },
                {
                    "name": "Duration:",
                    "value": "3"
                },
                {
                    "name": "Total:",
                    "value": "105"
                }
            ]
        },
        {
            "facts": [
                {
                    "name": "Total:",
                    "value": "105"
                },
                {
                    "name": "Fixable:",
                    "value": "20"
                }
            ],
            "text": "Overview"
        },        
        {
            "facts": [
                {
                    "name": "Critical:",
                    "value": "3"
                },
                {
                    "name": "High:",
                    "value": "3"
                },
                {
                    "name": "Low:",
                    "value": "70"
                },
                {
                    "name": "Negligible:",
                    "value": "50"
                }
            ],
            "text": "Summary"
        },
        {
            "facts": [
                {
                    "name": "Name:",
                    "value": "cbpm/cbpm-app"
                },
                {
                    "name": "Type:",
                    "value": "private"
                }
            ],
            "text": "Repository"
        }
    ],
    "potentialAction": [
        {
            "@type": "OpenUri",
            "name": "View in Harbor",
            "targets": [
                {
                    "os": "default",
                    "uri": "http://registry.dev.finworks.com"
                }
            ]
        }
    ]
}`
)

// MSTeamsHandler preprocess event data to msteams and start the hook processing
type MSTeamsHandler struct {
}

// Name ...
func (s *MSTeamsHandler) Name() string {
	return "MSTeams"
}

// Handle handles event to msteams
func (s *MSTeamsHandler) Handle(value interface{}) error {
	if value == nil {
		return errors.New("MSTeamsHandler cannot handle nil value")
	}

	event, ok := value.(*model.HookEvent)
	if !ok || event == nil {
		return errors.New("invalid notification msteams event")
	}

	return s.process(event)
}

// IsStateful ...
func (s *MSTeamsHandler) IsStateful() bool {
	return false
}

func (s *MSTeamsHandler) process(event *model.HookEvent) error {
	j := &models.JobData{
		Metadata: &models.JobMetadata{
			JobKind: job.KindGeneric,
		},
	}
	// Create a msteamsJob to send message to msteams
	j.Name = job.MSTeamsJob

	// Convert payload to msteams format
	payload, err := s.convert(event.Payload)
	if err != nil {
		return fmt.Errorf("convert payload to msteams body failed: %v", err)
	}

	j.Parameters = map[string]interface{}{
		"payload":          payload,
		"address":          event.Target.Address,
		"skip_cert_verify": event.Target.SkipCertVerify,
	}
	return notification.HookManager.StartHook(event, j)
}

func (s *MSTeamsHandler) convert(payLoad *model.Payload) (string, error) {
	data := make(map[string]interface{})
	data["Type"] = payLoad.Type
	data["OccurAt"] = payLoad.OccurAt
	data["Operator"] = payLoad.Operator
	eventData, err := json.MarshalIndent(payLoad.EventData, "", "\t")
	if err != nil {
		return "", fmt.Errorf("marshal from eventData %v failed: %v", payLoad.EventData, err)
	}
	data["EventData"] = "```" + strings.Replace(string(eventData), `"`, `\"`, -1) + "```"

	// DEBUG
	fmt.Print(data["EventData"])
	
	st, _ := template.New("msteams").Parse(MSTeamsBodyTemplate)
	var msteamsBuf bytes.Buffer
	if err := st.Execute(&msteamsBuf, data); err != nil {
		return "", fmt.Errorf("%v", err)
	}
	return msteamsBuf.String(), nil
}
