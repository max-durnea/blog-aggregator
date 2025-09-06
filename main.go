package main
import (
	"fmt"
	//"os"
	"github.com/max-durnea/blog-aggregator/internal/config"
)
type state struct{
	cfg Config*
}

type command struct{
	
}
func main(){
	
	cfg,err:=config.Read()
	if err!=nil {
		fmt.Println(err)
		return
	}
	err=cfg.SetUser("yagan")
	cfg,err = config.Read()
	fmt.Printf("%v\n",cfg)
}