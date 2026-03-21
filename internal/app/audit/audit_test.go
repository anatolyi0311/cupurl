package audit

import (
	"testing"

	"github.com/anatolyi0311/cupurl/internal/app/config"
)

type args struct {
	cfg    *config.ENVConfig
	userID int
}

func TestNewAudit(t *testing.T) {
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"ValidAuditFile", args{userID: 1, cfg: &config.ENVConfig{AuditFile: "audit/files/audit.json", AuditURL: ""}}, false},
		{"InvalidAuditFile", args{userID: 2, cfg: &config.ENVConfig{AuditFile: "", AuditURL: ""}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			aud := Audit{cfg: *tt.args.cfg, eventChan: make(chan Event, 1)}

			_, err := New(&aud.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewAudit() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// aud.eventChan = make(chan Event, 1)
			// aud.eventChan <- CreateEvent(2, "", "")
			// aud.Run(t.Context())

			aud.eventChan <- CreateEvent(3, "", "")
			ch := <-aud.eventChan
			if tt.wantErr == false && ch.UserID == tt.args.userID {
				t.Error("UserID equal, but no error was expected")
			}
			// aud.eventChan = make(chan Event, 1)
			aud.Update(CreateEvent(tt.args.userID, "", ""))
			ch = <-aud.eventChan
			if tt.wantErr == false && ch.UserID != tt.args.userID {
				t.Error("UserID not equal, but no error was expected")
			}
		})
	}
}
