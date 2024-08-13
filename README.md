# abbtr
abbtr comes as an option to the default "alias" command and it's a simple program to abbreviate long prompts in GNU/Linux terminal.
You can easily set rules, delete them, list them and update them with a clear list of parameters. It should be functional in any GNU/Linux distribution.

⭐ **FEATURES**

* Simplify your long commands in terminal

* Improve your times at making repetitive tasks

* You don't need to open any config file

* Store, list, update and delete rules

* Run your rules in bulk, i.e. `abbtr <rule1> <rule2>`

* Import rules from a local file

* Backup your rules to a local file

* Feeding bottles (adding variables inside a command)


:white_check_mark: **PROGRAMMING LANGUAGE**

abbtr is completly writen in Golang.


:ballot_box_with_check: **COMPILE YOURSELF**

If you preffer to compile yourself the source code run the following commands:

`git clone https://github.com/manuwarfare/abbtr.git`

`cd abbtr`

`go build -o abbtr`

`sudo cp abbtr /usr/bin/`

**Previous requirements to compile:** golang ('gcc-go', 'golang-bin')

To install Golang in your system run

  `sudo dnf install golang` or `sudo apt install golang` depending on your GNU/Linux distribution.


:question: **HOW IT FUNCTIONS?**

When you create a new rule it is saved as a bash script in ~/.local/bin and is executed every time you invoke it by its name.

:file_folder: **AFFECTED LOCATIONS**

 **~/.config/abbtr:** this directory is used to store the config file "abbtr.conf".

 **~/.local/share/abbtr:** this directory is used to store the registry log "abbtr.log".

 **~/.local/bin:** this directory is used to store the rule-scripts.

:pencil: **CREATING RULES**

First step after install the program is run `abbtr -h` to know about how the script functions. Some examples to create rules in a Fedora system terminal:

  `abbtr -n update "sudo dnf update -y && sudo dnf upgrade -y"` this long command will run after with only type `update`.

  `abbtr -n ssh "ssh user@example.com"` will connect to your SSH server only typing `ssh`

  Running a block of rules is as easy as run `abbtr <name1> <name2>`. This command will run two rules continuously but you can set as many as your implementation let.

:pencil: **IMPORTING RULES**

  `abbtr -i <file path>` will import rules from a local file.

  The path must to point to a file extension, i.e: .txt, .md, .html, etc.

  The stored rules must follow this syntax: `b:<rule> = <command>:b`

:pencil: **EXPORTING RULES**

  `abbtr -e` will start the backup assistant.

:pencil: **LISTING RULES**

There are two options to list the rules stored in abbtr.conf file.

  `abbtr -l` will list all the rules stored in abbtr.conf file.

  `abbtr -ln <name>` will list an specific rule.

:pencil: **REMOVING RULES**

  `abbtr -r <name>` will remove an specific rule.

  `abbtr -r a` will remove all rules stored in abbtr.conf.

:pencil: **FEEDING BOTTLES**

  The feeding bottles help you adding a variable inside a command. Use only one bottle for command.

  The feeding bottle syntax is this `b%('bottle_name')%b` and you can add it into any part of the command.

  Usage examples: `abbtr -n ssh "ssh -p 2222 b%('username')%b@example.com"`

  Execute the rule with: `ssh` and the system will prompt this:

  _The username is?:_

  If the credentials are valid, you will get connection via ssh to *example.com*.

  You can also predefine the value of a bottle at any time, this value will be automatically applied to all the rules when you run them in bulk, to do this use the next argument `-b=<variable:value>`.

  Usage examples: `abbtr -b=username:user1 ssh`

  This will run the next command: `ssh -p 2222 user1@example.com`


# 🤖 **TESTED ON**

🟢 Debian

🟢 Ubuntu

🟢 Linux Mint

🟢 MX Linux

🟢 Fedora

🟢 Almalinux

🟢 RockyLinux