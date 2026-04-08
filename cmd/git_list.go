// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Lucas José de Lima Silva

package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var gitListCmd = &cobra.Command{
	Use:   "list <commit>",
	Short: "Lista recursivamente os arquivos presentes em um commit",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		commit := args[0]
		out, err := exec.Command("git", "ls-tree", "-r", "--name-only", commit).CombinedOutput()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Erro ao consultar Git:\n%s\n", string(out))
			os.Exit(1)
		}
		
		// Direto para stdout. O usuário gerencia a paginação com | less
		fmt.Print(string(out))
	},
}

func init() {
	gitCmd.AddCommand(gitListCmd)
}
