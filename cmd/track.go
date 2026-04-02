package cmd

import (
	"fmt"
	"os"

	"spycode/internal/storage"

	"github.com/spf13/cobra"
)

var projectFlag string

var trackCmd = &cobra.Command{
	Use:   "track <caminho>",
	Short: "Adiciona um arquivo ou diretório ao monitoramento de um projeto",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		target := args[0]

		if _, err := os.Stat(target); os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Erro: O alvo '%s' não existe no disco.\n", target)
			os.Exit(1)
		}

		if err := storage.TrackPath(projectFlag, target); err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao rastrear alvo: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Alvo '%s' rastreado com sucesso no projeto '%s'.\n", target, projectFlag)
	},
}

func init() {
	trackCmd.Flags().StringVarP(&projectFlag, "project", "p", "", "Nome do projeto (obrigatório)")
	trackCmd.MarkFlagRequired("project")
	rootCmd.AddCommand(trackCmd)
}
