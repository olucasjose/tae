package cmd

import (
	"fmt"
	"os"

	"spycode/internal/storage"

	"github.com/spf13/cobra"
)

var untrackCmd = &cobra.Command{
	Use:   "untrack <caminho>",
	Short: "Remove um arquivo ou diretório do monitoramento de um projeto",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		target := args[0]

		if err := storage.UntrackPath(projectFlag, target); err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao remover alvo: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Alvo '%s' removido do projeto '%s'.\n", target, projectFlag)
	},
}

func init() {
	untrackCmd.Flags().StringVarP(&projectFlag, "project", "p", "", "Nome do projeto (obrigatório)")
	untrackCmd.MarkFlagRequired("project")
	rootCmd.AddCommand(untrackCmd)
}
