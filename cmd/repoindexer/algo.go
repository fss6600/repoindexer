package main

import "sort"

type dbt struct {
    id int
    path string
    size int
    mtime int
}

func up() {
    fsl := []string{"filesum.s","index.db","contr1.txt","arrange.txt","new.txt","folder//texttospeech.txt"}
    dbl := []dbt{
        dbt{1,"filesum.s", 123, 321},
        dbt{2,"index.db", 345,5432},
        dbt{2,"arrange.txt", 345,5432},
    }

    //var fi, di int

    sort.Slice(fsl, func(i,j int) bool {return fsl[i] < fsl[j]})
    sort.Slice(dbl, func(i,j int) bool {return dbl[i].path < dbl[j].path})
    for _, s := range fsl {
        print(s, " ")
    }

    for _, s := range dbl {
        print(s.path, " ")
    }

}

//func lessString(i, j) bool {
//    return a < b
//}

func main() {
    up()
}