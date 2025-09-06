package config
import(
	"os"
	//"fmt"
	"encoding/json"
)


type Config struct{
	DB_url string `json:"db_url"`
	CurrentUserName string `json:"current_user_name"`
}

const configFileName = ".gatorconfig.json"


func Read() (Config,error){

	homedir,err := os.UserHomeDir()
	if err != nil{
		return Config{}, err
	}

	file:=homedir+"/"+configFileName
	if err != nil{
		return Config{}, err
	}

	contents,err := os.ReadFile(file)
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	err = json.Unmarshal(contents,&cfg)
	if err != nil {
		return Config{}, err
	}
	return cfg,nil
}

func (cfg Config)SetUser(username string) error{
	cfg.CurrentUserName=username
	homedir,err:=os.UserHomeDir()
	if err != nil {
		return err
	}
	file:= homedir+"/"+configFileName
	js,err:= json.Marshal(cfg)
	if err := os.WriteFile(file, js,0666); err != nil {
		return err
	}
	return nil
}
