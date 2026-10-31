// Package domain holds the small shared vocabulary of stoa.
package domain

import (
	"fmt"
	"strings"
)

// NoteType is the kind of document in the vault.
type NoteType string

const (
	Note    NoteType = "note"
	Journal NoteType = "journal"
	Doc     NoteType = "doc"
)

const (
	FolderNote    = "notes"
	FolderJournal = "journal"
	FolderDoc     = "docs"
)

// Folder returns the vault subfolder for a type.
func (t NoteType) Folder() (string, error) {
	switch t {
	case Note:
		return FolderNote, nil
	case Journal:
		return FolderJournal, nil
	case Doc:
		return FolderDoc, nil
	default:
		return "", fmt.Errorf("unknown note type %q (want note, journal or doc)", string(t))
	}
}

// String implements fmt.Stringer.
func (t NoteType) String() string { return string(t) }

// ParseNoteType validates a user-supplied type.
func ParseNoteType(s string) (NoteType, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "note", "n":
		return Note, nil
	case "journal", "j":
		return Journal, nil
	case "doc", "d", "documentation":
		return Doc, nil
	default:
		return "", fmt.Errorf("unknown note type %q (want note, journal or doc)", s)
	}
}
