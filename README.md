# ./twgcals

TaskWarrior Google Calendar Sync

    GOBIN=/usr/local/bin/ go install ./cmd/twgcals

Now run ./twgcals regularly


Enjoy events synced to TaskWarrior.


## Details

The format of the added task is
* project:calendar name, use override to add . separators (ie my cal is named `personal.cal`)
* description:event summary, typically the title of the event
* due: start of event
* until: end of event

TL:DR; tasks clean themselves up
