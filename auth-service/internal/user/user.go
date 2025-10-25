// Komponen model user dan payload untuk registrasi dan login. Mulai dari model,repostitory, service, hingga handler.
package user

import "time"

type User struct {
	ID        int       `json:"id"`
	Email     string    `json:"email"`
	Password  string    `json:"-"` // Jangan kirim password di response JSON
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

//RegisterPayload adalah data yang kita harapkan dari request

type RegisterPayload struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

type LoginPayLoad struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}
