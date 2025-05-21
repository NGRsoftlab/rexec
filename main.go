package rexec

func main() {
	// envVars := make(map[string]string)
	// envVars["OS"] = "Unix"
	// envVars["KEEPER_PORT"] = "9183"
	// envVars["CLICKHOUSE_PORT"] = "9184"
	//
	// cmd := command.NewWithParser("[ -d $OS\\%s ] && echo '` + true + `' || echo '` + false + `'",
	// 	&parser.PathExistence{}, "some_dir")
	//
	// localConfig := config.NewLocalConfig()
	// localConfig.WithEnvVars(envVars)
	// localConfig.WithWorkDir("/home/user")
	//
	// var localPathExists bool
	// localSession := NewLocalSession(localConfig)
	// _, err := localSession.Run(context.Background(), cmd, &localPathExists)
	// if err != nil {
	// 	fmt.Printf("run error")
	// }
	//
	// sudoPassword := "SECRET_PASSWORD"
	//
	// cfg := config.NewConfig("user", "localhost", 22)
	// cfg.WithEnvVars(envVars).WithAgentAuth().WithPasswordAuth("password")
	//
	// privateKeyPath := filepath.Join(os.Getenv("HOME"), ".ssh", "id_rsa")
	// cfg.WithPrivateKeyPathAuth(privateKeyPath, "")
	//
	// privateKeyBytes, err := os.ReadFile(privateKeyPath)
	// if err != nil {
	// 	fmt.Printf("Err reading private key: %v\n", err)
	// }
	//
	// cfg.WithPrivateKeyBytesAuth(privateKeyBytes, "")
	//
	// cfg.WithSudoPassword(sudoPassword)

	// config, err := cfg.ClientConfig()
	// if err != nil {
	// 	fmt.Printf("Err creating config: %v\n", err)
	// }
	//
	//
	//
	// conn := rexec.Connect(cfg)
	//
	// sess := session.NewSession(client)
	// sess.Run()
	//
	// client.
}
