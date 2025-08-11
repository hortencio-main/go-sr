package main

/* a triangle */

import (
	"math"
    "fmt"
	"github.com/hortencio-main/go-sr"
)

type Vec3 struct {
    x, y, z float32
}

const (
    PI     = 3.14159                            //
    fov    = 45.0                               // field of view in degrees
    height = 24                                 // framebuffer height in characters
    width  = 80                                 // framebuffer width in characters
    aspect = float32(width*5)/float32(height*8) // aspect ratio (adjusted to 5:8, for a square output on the terminal)
    near   = 0.1                                // near plane
    far    = 100.0                              // far plane
)

var (
    buffer []byte
)

func init() {
    sr.Viewport(width, height)                 // create an framebuffer for the render
    buffer = make([]byte, height*(width + 1))  // create an buffer for the terminal output
    sr.PolygonMode(sr.FRONT_AND_BACK, sr.FILL) // set line drawing mode
}

func main() {
	var (
        top    float32 = near * float32(math.Tan(fov * PI / 360.0))
        bottom float32 =        -top
        right  float32 =  top*aspect
        left   float32 =      -right
        proj = sr.Frustum(left, right, bottom, top, near, far) // Set up perspective projection
    )

    camPos := Vec3{0,0,7}

    view := sr.LookAt( camPos.x, camPos.y, camPos.z, 0, 0, 0)
    sr.SetCamera(proj, view)
    sr.ClearColor(0,0,0)
    sr.Begin()
        for v := 0; v < len(cube); v+=4 {
            sr.Color3f( 1.0, 0.0, 0.0)
            sr.Vertex3f(cube[v  ][0],cube[v  ][1],cube[v  ][2])
            sr.Vertex3f(cube[v+1][0],cube[v+1][1],cube[v+1][2])
            sr.Vertex3f(cube[v+2][0],cube[v+2][1],cube[v+2][2])
            sr.Vertex3f(cube[v+3][0],cube[v+3][1],cube[v+3][2])
        }
    sr.End()
    
    image := sr.ReadPixels()
    for i := 0; i < height; i++ {
        for j := 0; j < width; j++ {
            if image[j + width*i][0] > .1 {
                buffer = append(buffer, '#')
            } else {
                buffer = append(buffer, ' ')
            }
        }
        buffer = append(buffer, '\n')
    }
    fmt.Println(string(buffer))
    buffer = buffer[:0:0]
}

var cube = [][3]float32{
    {-0.5,-0.5,0.0},
    { 0.5,-0.5,0.0},
    { 0.0, 0.5,0.0},
    { 0.0, 0.5,0.0}, // an extra vertice to draw an triangle
}
