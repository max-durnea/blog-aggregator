package config
import(
	"os"
	//"fmt"
	"encoding/json"
)

//A special struct to store the config file data which allows us to edit the file easily
type Config struct{
	DB_url string `json:"db_url"`
	CurrentUserName string `json:"current_user_name"`
}

const configFileName = ".gatorconfig.json"


func Read() (Config,error){
	//find the home directory because the config file is stored by default inside the home directory
	homedir,err := os.UserHomeDir()
	if err != nil{
		return Config{}, err
	}
	//build the path to the file
	file:=homedir+"/"+configFileName
	if err != nil{
		return Config{}, err
	}
	//read file contents as a string
	contents,err := os.ReadFile(file)
	if err != nil {
		return Config{}, err
	}
	//unmarshal the contents into a Config struct
	var cfg Config
	err = json.Unmarshal(contents,&cfg)
	if err != nil {
		return Config{}, err
	}
	return cfg,nil
}

func (cfg Config)SetUser(username string) error{
	//edit the config struct with the specified username
	cfg.CurrentUserName=username
	homedir,err:=os.UserHomeDir()
	if err != nil {
		return err
	}
	file:= homedir+"/"+configFileName
	//build the byte slice of the Config struct
	js,err:= json.Marshal(cfg)
	if err != nil {
		return err
	}
	//write the slice to the file 0666 is used to allow any user to read and write to the file but not execute it
	if err := os.WriteFile(file, js,0666); err != nil {
		return err
	}
	return nil
}
