package cmd

import (
	"fmt"
	"os"
)

// CompletionCmd generates shell completions.
type CompletionCmd struct {
	Bash CompletionBashCmd `cmd:"" help:"Generate bash completions"`
	Zsh  CompletionZshCmd  `cmd:"" help:"Generate zsh completions"`
	Fish CompletionFishCmd `cmd:"" help:"Generate fish completions"`
}

type CompletionBashCmd struct{}

func (c *CompletionBashCmd) Run() error {
	script := `_harvest_completions() {
    local cur="${COMP_WORDS[COMP_CWORD]}"
    local commands="version config auth completion time timer projects clients tasks users"

    if [ $COMP_CWORD -eq 1 ]; then
        COMPREPLY=($(compgen -W "$commands" -- "$cur"))
    fi
}

complete -F _harvest_completions harvest
`
	fmt.Fprint(os.Stdout, script)
	return nil
}

type CompletionZshCmd struct{}

func (c *CompletionZshCmd) Run() error {
	script := `#compdef harvest

_harvest() {
    local -a commands
    commands=(
        'version:Print version'
        'config:Manage configuration'
        'auth:Authentication and credentials'
        'completion:Generate shell completions'
        'time:Time entries'
        'timer:Timer commands'
        'projects:Projects'
        'clients:Clients'
        'tasks:Tasks'
        'users:Users'
    )

    _arguments \
        '1: :->command' \
        '*::arg:->args'

    case $state in
        command)
            _describe 'command' commands
            ;;
    esac
}

compdef _harvest harvest
`
	fmt.Fprint(os.Stdout, script)
	return nil
}

type CompletionFishCmd struct{}

func (c *CompletionFishCmd) Run() error {
	script := `complete -c harvest -f

complete -c harvest -n '__fish_use_subcommand' -a 'version' -d 'Print version'
complete -c harvest -n '__fish_use_subcommand' -a 'config' -d 'Manage configuration'
complete -c harvest -n '__fish_use_subcommand' -a 'auth' -d 'Authentication and credentials'
complete -c harvest -n '__fish_use_subcommand' -a 'completion' -d 'Generate shell completions'
complete -c harvest -n '__fish_use_subcommand' -a 'time' -d 'Time entries'
complete -c harvest -n '__fish_use_subcommand' -a 'timer' -d 'Timer commands'
complete -c harvest -n '__fish_use_subcommand' -a 'projects' -d 'Projects'
complete -c harvest -n '__fish_use_subcommand' -a 'clients' -d 'Clients'
complete -c harvest -n '__fish_use_subcommand' -a 'tasks' -d 'Tasks'
complete -c harvest -n '__fish_use_subcommand' -a 'users' -d 'Users'
`
	fmt.Fprint(os.Stdout, script)
	return nil
}
