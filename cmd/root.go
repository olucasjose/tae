package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "tae",
	Short: "Tae é um utilitário CLI para extração e empacotamento de código",
	Long: `Uma ferramenta modular para gerenciar, rastrear e extrair arquivos de tags de forma inteligente.

Dicas de Autocompletar:
Para habilitar o [TAB] no terminal, gere o script correspondente ao seu shell.
Exemplo Linux (Bash):
  tae completion bash | sudo tee /etc/bash_completion.d/tae > /dev/null
  # ou
  sudo sh -c 'tae completion bash > /etc/bash_completion.d/tae'
  exec bash

Exemplo Termux (Android):
  tae completion bash > /data/data/com.termux/files/usr/etc/bash_completion.d/tae
  exit (Reinicie a sessão do terminal completamente)`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
