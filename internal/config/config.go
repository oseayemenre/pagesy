package config

type Config struct {
	Cloudinary_cloud  string `mapstructure:"CLOUDINARY_CLOUD"`
	Cloudinary_key    string `mapstructure:"CLOUDINARY_KEY"`
	Cloudinary_secret string `mapstructure:"CLOUDINARY_SECRET"`
	Db_conn           string `mapstructure:"DB_CONN"`
}
