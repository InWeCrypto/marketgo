package marketgo

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dghubble/sling"
)

func TestNEO(t *testing.T) {
	client := sling.New()

	request, err := client.Get("http://localhost:5000/kline?symbol=NEO&currency=USDT&interval=5m").Request()

	assert.NoError(t, err)

	var data interface{}
	var failureV interface{}

	response, err := client.Do(request, &data, &failureV)

	if assert.NoError(t, err) {
		if response.StatusCode != http.StatusOK {
			printResult(failureV)
		} else {
			printResult(data)
		}
	}

}

func printResult(result interface{}) {

	data, _ := json.MarshalIndent(result, "", "\t")

	fmt.Println(string(data))
}
