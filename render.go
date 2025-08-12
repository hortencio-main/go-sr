package sr

import "math"


const PI = 3.1415926535897932384626433832795028841971693993751058209749445923078164062

type Color struct {
    r, g, b float32
}

type IVec2 struct {
    x, y int
}

type Vec3 struct {
	x, y, z float32
}

type Vec4 struct {
    x, y, z, w float32
}

type Quad struct {
    v [4]Vec4
    c Color
}

type Framebuffer struct {
    h, v int
    d    []Color
}

type Light struct {
    Pos  Vec3    // Position (for point lights)
    Dir  Vec3    // Direction (for directional lights)
    Color Color  // RGB intensity
    Type  int    // 0 = directional, 1 = point
    enabled bool
}

const (
    LIGHT_DIRECTIONAL = iota
    LIGHT_POINT
    
    FRONT_AND_BACK
    FRONT
    BACK
    
    FILL
    LINE
    POINT
    
    LIGHTING0
    LIGHTING1
    LIGHTING2
    LIGHTING3
    
    POSITION
    DIFFUSE
)

var (
    framebuffer Framebuffer
    zBuffer         []float32
    MatrixModelView [16]float32
    Submit  Quad
    SubmitI int
    SubmitC Color
    polygonModeFront int
    polygonModeBack  int
    Lights = [4]Light{}
    
    Vertices    []IVec2
    Colors      []Color
    Depth       []float32
    lastVertex  *IVec2
)

func Viewport(h, v int) {
    framebuffer = Framebuffer{
        h: h,
        v: v,
        d: make([]Color, h*v),
    }
    zBuffer = make([]float32, h*v)
}

func XY() (int,int) {
    return framebuffer.h, framebuffer.v
}

func PolygonMode(face, mode int) {
    switch face {
    case FRONT:
        polygonModeFront = mode
    case BACK:
        polygonModeBack = mode
    case FRONT_AND_BACK:
        polygonModeFront = mode
        polygonModeBack = mode
    }
}

func Enable(v int) {
    switch v {
    case LIGHTING0: Lights[0].enabled = true
    case LIGHTING1: Lights[1].enabled = true
    case LIGHTING2: Lights[2].enabled = true
    case LIGHTING3: Lights[3].enabled = true
    }
}

func Disable(v int) {
    switch v {
    case LIGHTING0: Lights[0].enabled = false
    case LIGHTING1: Lights[1].enabled = false
    case LIGHTING2: Lights[2].enabled = false
    case LIGHTING3: Lights[3].enabled = false
    }
}

func Lightfv(id, attribute int, value []float32) {

    var selectedLight *Light
    switch id {
    case LIGHTING0: selectedLight = &Lights[0]
    case LIGHTING1: selectedLight = &Lights[1]
    case LIGHTING2: selectedLight = &Lights[2]
    case LIGHTING3: selectedLight = &Lights[3]
    default:
        panic("Invalid light ID")
    }
    switch attribute {
    case POSITION:
        if value[3] > 0.99 {
            selectedLight.Type = LIGHT_DIRECTIONAL
            selectedLight.Dir = Vec3{value[0],value[1],value[2]}
        } else {
            selectedLight.Type = LIGHT_POINT
            selectedLight.Pos = Vec3{value[0],value[1],value[2]}
        }
    case DIFFUSE:
        selectedLight.Color = Color{value[0],value[1],value[2]}
    default:
        panic("Invalid light attribute")
    }
}

func Vertex3f(x, y, z float32) {
    Submit.v[SubmitI] = Vec4{x, y, z, 1}

    if SubmitI++; SubmitI < 4 { // Wait until we have 4 vertices
        return
    }

    Submit.c = SubmitC
    quad := Submit

    var screenCoordsx [4]int
    var screenCoordsy [4]int
    var transformedVerts [4]Vec4

    for j := 0; j < 4; j++ {
        transformed := transformVertex(quad.v[j], MatrixModelView)
        transformedVerts[j] = transformed
        screenCoordsx[j], screenCoordsy[j] = viewportTransform(perspectiveDivide(transformed))
    }

    v0 := transformedVerts[0] // Per-face lighting
    v1 := transformedVerts[1] // Face normal in view space
    v2 := transformedVerts[2]

    edge1 := Vec3{v1.x - v0.x, v1.y - v0.y, v1.z - v0.z}
    edge2 := Vec3{v2.x - v0.x, v2.y - v0.y, v2.z - v0.z}
    normal := normalize(cross(edge1, edge2))
    
    var totalR, totalG, totalB float32
    base := Submit.c
    
    enabledLights := false
    
    for _, light := range Lights {
        if !light.enabled {
            continue
        }
        enabledLights = true
        
        var L Vec3

        switch(light.Type) {
        case LIGHT_DIRECTIONAL:
            L = normalize(light.Dir)
        case LIGHT_POINT:
            // Light vector from face center to light
            center := Vec3{
                (v0.x + v1.x + v2.x + transformedVerts[3].x) * 0.25,
                (v0.y + v1.y + v2.y + transformedVerts[3].y) * 0.25,
                (v0.z + v1.z + v2.z + transformedVerts[3].z) * 0.25,
            }
            L = normalize(Vec3{
                light.Pos.x - center.x,
                light.Pos.y - center.y,
                light.Pos.z - center.z,
            })
        default:
            panic(1)
        }

        
        diff := dot(normal, L) // Diffuse term
        if diff < 0 {
            diff = 0
        }

        totalR += base.r * light.Color.r * diff // Add contribution from this light
        totalG += base.g * light.Color.g * diff
        totalB += base.b * light.Color.b * diff
    }
    
    var color Color
    if enabledLights {
        if totalR > 1 { totalR = 1 } // Clamp to [0,1]
        if totalG > 1 { totalG = 1 }
        if totalB > 1 { totalB = 1 }
        color = Color{r: totalR, g: totalG, b: totalB}
    } else {
        color = base
    }
    
    var dist float32
    for j := 0; j < 4; j++ {
        dx := transformedVerts[j].x
        dy := transformedVerts[j].y
        dz := transformedVerts[j].z
        dist += dx*dx + dy*dy + dz*dz
    }
    dist /= 4 // average distance

    i := screenCoordsx
    j := screenCoordsy
    distance := dist

    switch polygonModeFront {
    case LINE:
        v0 := IVec2{i[0], j[0]}
        v1 := IVec2{i[1], j[1]}
        v2 := IVec2{i[2], j[2]}
        v3 := IVec2{i[3], j[3]}
        Vertices = append(Vertices, drawLine(v0, v1, distance)...)
        Vertices = append(Vertices, drawLine(v1, v2, distance)...)
        Vertices = append(Vertices, drawLine(v2, v3, distance)...)
        Vertices = append(Vertices, drawLine(v3, v0, distance)...)
    case POINT:
        for k := 0; k < 4; k++ {
            curr := IVec2{i[k], j[k]}
            Vertices = append(Vertices, curr)
            Colors = append(Colors, color)
            Depth = append(Depth, distance)
        }
    case FILL:
        v := [4]IVec2{
            {i[0], j[0]},
            {i[1], j[1]},
            {i[2], j[2]},
            {i[3], j[3]},
        }
        Vertices = append(Vertices, fillTriangle(v[0], v[1], v[2], distance, color)...) // v0-v1-v2
        Vertices = append(Vertices, fillTriangle(v[0], v[2], v[3], distance, color)...) // v0-v2-v3
    }

    lastVertex = nil
    SubmitI = 0
}




func Translatef(f []float32){
    
}


func Rotatef(f []float32){
    if len(f) < 4 {
        panic("Rotatef requires 4 values: angle (deg), x, y, z")
    }
    angle := f[0] * (PI / 180.0) // degrees to radians
    x, y, z := f[1], f[2], f[3]
    length := float32(math.Sqrt(float64(x*x + y*y + z*z))) // normalize axis
    x /= length
    y /= length
    z /= length
    
    
    c := float32(math.Cos(float64(angle)))
    s := float32(math.Sin(float64(angle)))
    ic := 1 - c
    r := [16]float32{
        c + x*x*ic,   y*x*ic+z*s, z*x*ic-y*s, 0,
        x*y*ic-z*s,   c + y*y*ic, z*y*ic+x*s, 0,
        x*z*ic+y*s,   y*z*ic-x*s, c + z*z*ic, 0,
        0,            0,          0,          1,
    }
    MatrixModelView = MultMatrix(MatrixModelView, r)
}

//~ func Blend

// yet to decide an portable way of handling colors
// an 16 color terminal view would have 8 colors with 3 levels of brightness
func Color3f(r, g, b float32) {
    SubmitC = Color{r, g, b}
}

// we only intend on supporting quads
// for now Begin doesn't do anything
func Begin() { }

// End pops all the quads we've created to the framebuffer
func End() {
    mx, my  := XY()
    for k, v := range Vertices {
        if (v.x > 0) && (v.x < mx) && (v.y > 0) && (v.y < my) {
            index := v.x + mx*v.y
            if Depth[k] < zBuffer[index] {
                framebuffer.d[index] = Colors[k]
                zBuffer[index] = Depth[k]
            }
        }
    }
}

func ReadPixels() (image [][3]float32) {
    mx, my  := XY()
    for i := 0; i < my; i++ {
        for j := 0; j < mx; j++ {
            fbcolor := framebuffer.d[j+i*mx]
            image = append(image, [3]float32{
                fbcolor.r,
                fbcolor.g,
                fbcolor.b,
            })
        }
    }
    return image
}

func ClearColor(r, g, b float32) {
    mx, my  := XY()
    for i := 0; i < my; i++ {
        for j := 0; j < mx; j++ {
            framebuffer.d[j+i*mx] = Color{r,g,b}
            zBuffer[j+i*mx] = 999999999.0
        }
    }
    Vertices = Vertices[:0:0]
    Colors   =   Colors[:0:0]
    Depth    =    Depth[:0:0]
}

func transformVertex(v Vec4, m [16]float32) Vec4 {
    return Vec4{
        x: v.x*m[0] + v.y*m[4] + v.z*m[8]  + v.w*m[12],
        y: v.x*m[1] + v.y*m[5] + v.z*m[9]  + v.w*m[13],
        z: v.x*m[2] + v.y*m[6] + v.z*m[10] + v.w*m[14],
        w: v.x*m[3] + v.y*m[7] + v.z*m[11] + v.w*m[15],
    }
}

func viewportTransform(v Vec3) (x, y int) {
    x = int((v.x + 1.0) * 0.5 * float32(framebuffer.h))
    y = int((1.0 - (v.y + 1.0) * 0.5) * float32(framebuffer.v)) // flip Y axis for screen
    return
}

func perspectiveDivide(v Vec4) Vec3 {
    return Vec3{
        v.x/v.w,
        v.y/v.w,
        v.z/v.w,
    }
}

func abs(f int) int {
    if f < 0 {
        return -f
    }
    return f
}
func absf(f float32) float32 {
    if f < 0 {
        return -f
    }
    return f
}

func normalize(v Vec3) Vec3 {
    len := float32(math.Sqrt(float64(v.x*v.x + v.y*v.y + v.z*v.z)))
    return Vec3{v.x / len, v.y / len, v.z / len}
}

func cross(a, b Vec3) Vec3 {
	return Vec3{
		a.y*b.z - a.z*b.y,
		a.z*b.x - a.x*b.z,
		a.x*b.y - a.y*b.x,
	}
}

func dot(a, b Vec3) float32 {
	return a.x*b.x + a.y*b.y + a.z*b.z
}

func LookAt(a1, a2, a3, b1, b2, b3 float32) [16]float32 {
    
    eye := Vec3{a1, a2, a3}
    center := Vec3{b1, b2, b3}
    
	up := Vec3{0, 1, 0}

	z := normalize(Vec3{
		x: eye.x - center.x,
		y: eye.y - center.y,
		z: eye.z - center.z,
	})
	x := normalize(cross(up, z))
	y := cross(z, x)

	return [16]float32{
		x.x, y.x, z.x, 0,
		x.y, y.y, z.y, 0,
		x.z, y.z, z.z, 0,
		-dot(x, eye), -dot(y, eye), -dot(z, eye), 1,
	}
}

func Frustum(left, right, bottom, top, near, far float32) [16]float32 {
	return [16]float32{
		(2.0*near)/(right-left),                    0,(right+left)/(right-left),                                0,
                              0,(2*near)/(top-bottom),(top+bottom)/(top-bottom),                                0,
                              0,                    0,   -(far+near)/(far-near), -2.0 * far * near / (far - near),
                              0,                    0,                       -1,                                0,
	}
}

func MultMatrix(a, b [16]float32) [16]float32 {
	var r [16]float32
	for row := 0; row < 4; row++ {
		for col := 0; col < 4; col++ {
			sum := float32(0)
			for k := 0; k < 4; k++ {
				sum += a[k*4+row] * b[col*4+k]
			}
			r[col*4+row] = sum
		}
	}
	return r
}

func SetCamera(projection, view [16]float32) {
    MatrixModelView = MultMatrix(projection, view)
}

// Bresenham line drawing algorithm
func drawLine(a, b IVec2, distance float32) []IVec2 {
    var points []IVec2

    x0, y0 := a.x, a.y
    x1, y1 := b.x, b.y

    dx :=  abs(x1 - x0)
    dy := -abs(y1 - y0)

    sx := 1
    if x0 >= x1 {
        sx = -1
    }

    sy := 1
    if y0 >= y1 {
        sy = -1
    }

    err := dx + dy
    for {
        if (x0 < 0) || (x0 > (framebuffer.h-1)) || (y0 < 0) || (y0 > (framebuffer.v-1)) {
            break
        }
		
        points = append(points, IVec2{x0, y0})
        
        Colors = append(Colors, SubmitC) // temporary: color the vertex
        
        Depth = append(Depth, distance)
        
        
        if x0 == x1 && y0 == y1 {
            break
        }
        e2 := 2 * err
        if e2 >= dy {
            err += dy
            x0 += sx
        }
        if e2 <= dx {
            err += dx
            y0 += sy
        }
    }
    
    return points
}


func fillTriangle(v0, v1, v2 IVec2, distance float32, color Color) []IVec2 {
    var result []IVec2

    // Sort vertices by y-coordinate ascending (v0.y <= v1.y <= v2.y)
    if v1.y < v0.y {
        v0, v1 = v1, v0
    }
    if v2.y < v0.y {
        v0, v2 = v2, v0
    }
    if v2.y < v1.y {
        v1, v2 = v2, v1
    }

    edgeInterpolate := func(y0, y1, x0, x1 int) []int {
        var result []int
        dy := y1 - y0
        if dy == 0 {
            return []int{x0}
        }
        dx := x1 - x0
        for i := 0; i <= dy; i++ {
            x := x0 + dx*i/dy
            result = append(result, x)
        }
        return result
    }

    // Interpolate edges
    x01 := edgeInterpolate(v0.y, v1.y, v0.x, v1.x)
    x12 := edgeInterpolate(v1.y, v2.y, v1.x, v2.x)
    x02 := edgeInterpolate(v0.y, v2.y, v0.x, v2.x)

    x012 := append(x01[:len(x01)-1], x12...)

    yStart := v0.y
    yEnd := v2.y

    for y := yStart; y <= yEnd; y++ {
        i := y - yStart
        var xa, xb int
        if i < len(x02) && i < len(x012) {
            xa, xb = x02[i], x012[i]
            if xa > xb {
                xa, xb = xb, xa
            }
            for x := xa; x <= xb; x++ {
                result = append(result, IVec2{x, y})
                Colors = append(Colors, color)  // temporary: color the vertex
                Depth = append(Depth, distance) // temporary: add distance for the z buffer
            }
        }
    }
    return result
}

