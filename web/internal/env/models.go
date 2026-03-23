package env

import "time"

type Application struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type Environment struct {
	ID        int64     `json:"id"`
	AppID     int64     `json:"app_id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type Variable struct {
	ID        int64     `json:"id"`
	EnvID     int64     `json:"env_id"`
	Key       string    `json:"key"`
	Value     string    `json:"value,omitempty"`
	Version   int64     `json:"version"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type VariableVersion struct {
	ID        int64     `json:"id"`
	VarID     int64     `json:"var_id"`
	EnvID     int64     `json:"env_id"`
	Key       string    `json:"key"`
	Value     string    `json:"value,omitempty"`
	Version   int64     `json:"version"`
	CreatedAt time.Time `json:"created_at"`
}
