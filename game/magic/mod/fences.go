package mod

// The fences of farmland (combat/fences.go). Pictures of the replacement folder:
//
//   environment/Farmland/prop_fence_1.png, prop_fence_2.png, ...
//
// from 1 on without a gap. A fence stands along a side of a tile, and the sides of a tile run two
// ways on the screen: up to the right, and down to the right. Which way a picture runs is read
// from the picture itself: from where its pixels lie.

import (
    "fmt"
    "image"
    "image/png"
    "os"
    "path/filepath"
)

const fenceName = "prop_fence"
const maxFences = 99

func FenceFile(number int) string {
    return fmt.Sprintf("%v_%v.png", fenceName, number)
}

// a picture of a folder of the environment by the name of its file. nil if it is not there
func ReadNamed(set string, name string) image.Image {
    if folder == "" || set == "" {
        return nil
    }
    path := filepath.Join(folder, environmentFolder, set, name)
    if !hasFile(path) {
        return nil
    }

    file, err := os.Open(path)
    if err != nil {
        return nil
    }
    defer file.Close()

    picture, err := png.Decode(file)
    if err != nil {
        reportOnce(fmt.Sprintf("Replacement picture %v can not be read: %v", path, err))
        return nil
    }
    return picture
}

// true if what a picture shows runs up to the right: its pixels further right lie higher
func RunsUp(picture image.Image) bool {
    bounds := picture.Bounds()
    var count, sumX, sumY float64
    for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
        for x := bounds.Min.X; x < bounds.Max.X; x++ {
            _, _, _, alpha := picture.At(x, y).RGBA()
            if alpha >= 0x8000 {
                count += 1
                sumX += float64(x)
                sumY += float64(y)
            }
        }
    }
    if count == 0 {
        return false
    }

    middleX := sumX / count
    middleY := sumY / count
    together := 0.0
    for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
        for x := bounds.Min.X; x < bounds.Max.X; x++ {
            _, _, _, alpha := picture.At(x, y).RGBA()
            if alpha >= 0x8000 {
                together += (float64(x) - middleX) * (float64(y) - middleY)
            }
        }
    }
    // down the screen is more
    return together < 0
}

// the numbers of the fences of a folder: the ones that run up to the right, and the ones that run
// down to the right
func Fences(set string) ([]int, []int) {
    var up, down []int
    if folder == "" || set == "" {
        return up, down
    }
    delete(folderLists, filepath.Join(folder, environmentFolder, set))

    for number := 1; number <= maxFences; number++ {
        picture := ReadNamed(set, FenceFile(number))
        if picture == nil {
            break
        }
        if RunsUp(picture) {
            up = append(up, number)
        } else {
            down = append(down, number)
        }
    }
    return up, down
}
