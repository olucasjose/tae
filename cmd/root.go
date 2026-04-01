package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "spycode",
	Short: "Spycode é um utilitário CLI para extração e empacotamento de código",
	Long: `Uma ferramenta modular para gerenciar, rastrear e extrair arquivos de projetos de forma inteligente.

Dicas de Autocompletar:
Para habilitar o [TAB] no terminal, gere o script correspondente ao seu shell.
Exemplo Linux (Bash):
  sudo spycode completion bash -o /etc/bash_completion.d/spycode
  exec bash

Exemplo Termux (Android):
  spycode completion bash > /data/data/com.termux/files/usr/etc/bash_completion.d/spycode
  exit (Reinicie a sessão do terminal completamente)`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
