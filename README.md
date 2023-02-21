## Overview
go-llca is a life-like cellular automaton simulation tool written in Go. It is built to allow **easy experimentation** with different automaton rules while being **very performant**: go-llca is multithreaded and can smoothly simulate boards with millions of cells. 

go-llca also makes it possible to export crisp GIFs of the performed simulations.

![out2](https://user-images.githubusercontent.com/92261790/220476014-adcc6500-c2b1-4e34-8750-baddff8e43a9.gif)


## What are life-like cellular automata?

The most famous example of an LLCA is Conway's Game of Life: a 2D grid of cells, each of which is either dead or alive. The grid is repeatedly updated based on very simple rules:
1. Any dead cell with exactly 3 live neighbors becomes a live cell (birth rule).
2. Any live cell with exactly 2 or 3 live neighbors survives (survival rule).
3. All other live cells die in the next generation. Similarly, all other dead cells stay dead.

Life-like cellular automata are just like that, except we allow different birth and survival rules. For example, we may have an LLCA where a live cell is born from any dead cell which has 2, 3, 4, or 7 live neighbors, and any live cell survives if it has 3 or 5 live neighbors. We can then concisely write this rule as B2347/S35.

These sorts of simple rules, when iterated many times, can give rise to surprisingly complex patterns. Examples generated using go-llca are shown in the gallery below.

## Installation
To clone and build (requires Go 1.18 or newer):
```bash
$ git clone https://github.com/fplonka/go-llca.git
$ cd go-llca
$ go build .
```
Then run with:
```bash
$ ./go-llca
```

# Gallery
**WARNING**: Some of the following animations contain **rapidly flashing lights**.

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
