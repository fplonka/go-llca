## Overview
go-llca is a life-like cellular automaton simulation tool written in Go. It is built to allow **fast and easy experimention** with different automaton rules while being very **performant**: go-llca is multithreaded and can smoothly simulate boards with millions of cells.

![out2](https://user-images.githubusercontent.com/92261790/220154420-ab0629ab-3e55-4c66-8ace-e1fad030316e.gif)

## Installation
To fetch, build and install to $GOPATH/bin (requires Go 1.18 or newer):
```bash
go install github.com/fplonka/go-llca@latest
```
Then to run (assuming default $GOPATH):
```bash
~/go/bin/go-llca
```

## Gallery
B3/S23 (Conway's Game of Life):

![B3/S23](https://user-images.githubusercontent.com/92261790/220101889-3bad143a-91dd-4b35-9eb0-fcb8174a24ed.gif)

B34/S2678:

![B34/S2678](https://user-images.githubusercontent.com/92261790/220105006-f3a1b45e-e7cd-402c-91e1-2f812e39fd4c.gif)

B4678/S35678:

![B4678/S35678](https://user-images.githubusercontent.com/92261790/220102296-48009cd2-c48e-4b41-a7e9-46ac05d3a46d.gif)

B45678/S2345:

![B45678/S2345](https://user-images.githubusercontent.com/92261790/220122046-d14eba4d-cdaf-4343-a3f3-12fd73ad9f20.gif)

B2/S678:

![B2/S678](https://user-images.githubusercontent.com/92261790/220112009-4f93ec16-20a9-468d-8c24-758705ab5cfc.gif)

B23/S1:

![B23/S1](https://user-images.githubusercontent.com/92261790/220104000-e936b3ab-8eda-4086-a049-c859e885a53a.gif)

B34/S234567:

![B34/S234567](https://user-images.githubusercontent.com/92261790/220106344-fb777376-ed81-442c-aaa6-0ce04b278881.gif)

B378/S245678:

![B378/S245678](https://user-images.githubusercontent.com/92261790/220110026-18c21344-df6e-4591-bd03-cb09caaed2b7.gif)

B3578/S24678:

![B3578/S24678](https://user-images.githubusercontent.com/92261790/220169257-7645fa04-58e3-4293-aa8e-3c6e265aceab.gif)
