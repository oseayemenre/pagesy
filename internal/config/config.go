package config

type Config struct {
	Cloudinary_cloud     string `mapstructure:"CLOUDINARY_CLOUD"`
	Cloudinary_key       string `mapstructure:"CLOUDINARY_KEY"`
	Cloudinary_secret    string `mapstructure:"CLOUDINARY_SECRET"`
	Db_conn              string `mapstructure:"DB_CONN"`
	Google_client_id     string `mapstructure:"GOOGLE_CLIENT_ID"`
	Google_client_secret string `mapstructure:"GOOGLE_CLIENT_SECRET"`
	Session_secret       string `mapstructure:"SESSION_SECRET"`
	Jwt_secret           string `mapstructure:"JWT_SECRET"`
	Host                 string `mapstructure:"HOST"`
}
