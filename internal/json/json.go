package json

import (
	"encoding/json"
	"net/http"

	"github.com/go-playground/validator/v10"
)

// Inisialisasi validator (hanya dibuat satu kali untuk seluruh aplikasi)
var Validate = validator.New()

func Write(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func Read(r *http.Request, data any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(data); err != nil {
		return err
	}

	// Tambahkan proses validasi struct secara otomatis
	if err := Validate.Struct(data); err != nil {
		return err
	}

	return nil
}
