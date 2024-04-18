package processclienttype

import (
	"fmt"
	"io"
	"net/http"
)

type ProcessClient struct {
	ServerURL string
}

func (c *ProcessClient) ConfigureClientDefault() {
	c.ServerURL = "http://kk-crossplane3:12345"
}

func (c *ProcessClient) ConfigureClient(serverURL string) {
	c.ServerURL = serverURL
}

func (c *ProcessClient) CreaProcesso(id string) {
	// Crea una nuova richiesta POST con l'ID nell'URL
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/processo/%s", c.ServerURL, id), nil)
	if err != nil {
		fmt.Println("Errore durante la creazione della richiesta POST:", err)
		return
	}

	// Invia la richiesta POST al server
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Errore durante l'invio della richiesta POST:", err)
		return
	}
	defer resp.Body.Close()

	// Leggi la risposta dal server
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Errore durante la lettura della risposta:", err)
		return
	}

	fmt.Println("\nRisposta dal server:\n" + string(responseBody))
}

func (c *ProcessClient) EliminaProcesso(id string) {
	// Crea una nuova richiesta DELETE con l'ID nell'URL
	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/processo/%s", c.ServerURL, id), nil)
	if err != nil {
		fmt.Println("Errore durante la creazione della richiesta DELETE:", err)
		return
	}

	// Invia la richiesta DELETE al server
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Errore durante l'invio della richiesta DELETE:", err)
		return
	}
	defer resp.Body.Close()

	// Leggi la risposta dal server
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Errore durante la lettura della risposta:", err)
		return
	}

	fmt.Println("\tRisposta dal server:" + string(responseBody))
}
