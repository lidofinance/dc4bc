package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	singleID     = "ce351963-6e7e-4bb7-8300-fe3f7af931b0"
	singleOffset = "0"
	spanOffset   = "0-15"
	listIDs      = "ce351963-6e7e-4bb7-8300-fe3f7af931b0,e7f1c533-2879-4c26-adbf-ab2e89b1ed16,4f750830-be62-4e20-8045-0e344a2519d9"
	listOffset   = "1,2,3,4,5,9,15,20"
)

var (
	singleIDResultMessages     = []string{"ce351963-6e7e-4bb7-8300-fe3f7af931b0"}
	singleOffsetResultMessages = []string{"0"}
	spanOffsetResultMessages   = []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11", "12", "13", "14", "15"}
	listIDsResultMessages      = []string{"ce351963-6e7e-4bb7-8300-fe3f7af931b0", "e7f1c533-2879-4c26-adbf-ab2e89b1ed16", "4f750830-be62-4e20-8045-0e344a2519d9"}
	listOffsetResultMessages   = []string{"1", "2", "3", "4", "5", "9", "15", "20"}

	results = map[string][]string{
		singleID:     singleIDResultMessages,
		singleOffset: singleOffsetResultMessages,
		spanOffset:   spanOffsetResultMessages,
		listIDs:      listIDsResultMessages,
		listOffset:   listOffsetResultMessages,
	}
)

func TestParseMessagesToIgnore(t *testing.T) {
	suits := []string{singleID, singleOffset, spanOffset, listIDs, listOffset}

	t.Run("SingleMessage", func(t *testing.T) {
		for _, suit := range suits {
			match := messagesToIgnoreSingleRx.MatchString(suit)
			if suit == singleID || suit == singleOffset {
				assert.Truef(t, match, "single ID or offset should contain a match for the relative rx")
				result, err := parseMessagesToIgnore(suit)
				assert.Equal(t, results[suit], result)
				assert.Nil(t, err)
			} else if match {
				t.Logf("only single ID or offset should contain a match for the relative tx")
				t.Logf("test failed for suit %s", suit)
			}
		}
	})
	t.Run("MessagesSpan", func(t *testing.T) {
		for _, suit := range suits {
			match := messagesToIgnoreSpanRx.MatchString(suit)
			if suit == spanOffset {
				assert.Truef(t, match, "span of message offsets should contain a match for the relative rx")
				result, err := parseMessagesToIgnore(suit)
				assert.Equal(t, results[suit], result)
				assert.Nil(t, err)
			} else if match {
				t.Logf("only span of message offsets should contain a match for the relative tx")
				t.Logf("test failed for suit %s", suit)
			}
		}
	})
	t.Run("ListOfMessages", func(t *testing.T) {
		for _, suit := range suits {
			match := messagesToIgnoreListRx.MatchString(suit)
			if suit == listIDs || suit == listOffset {
				assert.Truef(t, match, "list of IDs or offsets should contain a match for the relative rx")
				result, err := parseMessagesToIgnore(suit)
				assert.Equal(t, results[suit], result)
				assert.Nil(t, err)
			} else if match {
				t.Logf("list of IDs or offsets should contain a match for the relative tx")
				t.Logf("test failed for suit %s", suit)
			}
		}
	})
}