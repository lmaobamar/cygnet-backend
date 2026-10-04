package models

import "github.com/google/uuid"

func newID(id *uuid.UUID) error {
	if *id != uuid.Nil {
		return nil
	}
	v, err := uuid.NewV7()
	if err != nil {
		return err
	}
	*id = v
	return nil
}
