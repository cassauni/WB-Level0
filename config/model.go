package config

type ConfigModel struct {
	HTTP struct {
		Addr string 
	}

	Postgres struct {
		DSN string
	}

	Kafka struct {
		Brokers []string 
		Topic   string
		GroupID string
	}

	Redis struct {
		Addr       string 
		Password   string
		DB         int
		TTLSeconds int    
		KeyPrefix  string 
	}
}
