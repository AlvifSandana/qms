package worker

import "testing"

func TestRenderTemplate(t *testing.T) {
	payload := payloadData{
		"ticket_number": "A-001",
		"counter_id":    "C-01",
	}
	template := "Ticket {ticket_number} to counter {counter_id}"
	got := renderTemplate(template, payload)
	if got != "Ticket A-001 to counter C-01" {
		t.Fatalf("unexpected template render: %s", got)
	}
}
