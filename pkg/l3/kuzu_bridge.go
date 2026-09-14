package l3

import (
	"bytes"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// KuzuBridge permite al plano L3 consultar el grafo PFO de Kùzu
type KuzuBridge struct {
	ManagerScriptPath string
}

// NewKuzuBridge inicializa el conector del grafo
func NewKuzuBridge(scriptPath string) *KuzuBridge {
	return &KuzuBridge{
		ManagerScriptPath: scriptPath,
	}
}

// QueryCypher ejecuta una consulta openCypher a través del gestor PFO
func (kb *KuzuBridge) QueryCypher(query string) (string, error) {
	var cmd *exec.Cmd

	if runtime.GOOS == "windows" {
		// En Windows se canaliza mediante WSL donde corre Kùzu DB
		cypherEscaped := strings.ReplaceAll(query, "\"", "\\\"")
		cmd = exec.Command("wsl", "python3", "scripts/kuzu_pfo_manager.py", "--cypher", cypherEscaped)
	} else {
		cmd = exec.Command("python3", kb.ManagerScriptPath, "--cypher", query)
	}

	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("error ejecutando consulta Cypher en Kùzu: %w (stderr: %s)", err, errBuf.String())
	}

	return outBuf.String(), nil
}

// GetAxiomaticStatus consulta el resumen de Principios y Funciones del grafo
func (kb *KuzuBridge) GetAxiomaticStatus() (string, error) {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("wsl", "python3", "scripts/kuzu_pfo_manager.py", "--verify")
	} else {
		cmd = exec.Command("python3", kb.ManagerScriptPath, "--verify")
	}

	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("error verificando grafo: %w (%s)", err, errBuf.String())
	}

	return outBuf.String(), nil
}
