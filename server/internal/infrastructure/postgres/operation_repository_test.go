package postgres

import (
	"testing"
	"time"

	"github.com/zhouwu97/key-cabinet/server/internal/repository"
)

func TestOperationPersistenceValuesUsesNullForOptionalIDs(t *testing.T) {
	operation := &repository.DeviceOperation{
		ID:             "op_test",
		RequestID:      "req_test",
		UserID:         "usr_test",
		BorrowRecordID: "bor_test",
		DeviceID:       "dev_test",
		KeyID:          "key_test",
		Action:         "RETURN",
		Status:         "EXECUTING",
		CreatedAt:      time.Now().UTC(),
	}

	values := operationPersistenceValues(operation)
	for _, field := range []string{"reservation_id", "slot_id"} {
		if values[field] != nil {
			t.Fatalf("%s = %#v, want NULL", field, values[field])
		}
	}
	if values["borrow_record_id"] != "bor_test" {
		t.Fatalf("borrow_record_id = %#v, want bor_test", values["borrow_record_id"])
	}
}
