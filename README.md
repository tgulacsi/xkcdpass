# xkcdpass
[Password strength vs. human memory](https://xkcd.com/936/)

TL;DR; Use a long _passphrase_ instead of a short gibberish with symbols, numbers and mixed case.

## Install
I don't want to redistribute the word lists for legal reasons, the program can do it
upon installing.

First, check out the source to somewhere:

    git clone https://github.com/tgulacsi/xkcdpass.git
    cd xkcdpass
    go generate  # downloads the word lists
    go install 

## Usage
Default it generates four words according to `LANG` env var.

    xkcdpass
    afford identification joy perhaps

The number of words can be given as an argument:

    xkcdpass 11
    contribution couple dead flower folk gradually hole leading offense station tobacco 

And the LANG env var can be overridden with the `-lang` flag:

    xkcdpass -lang=french 5
    boîte terre venir arriver baignade

