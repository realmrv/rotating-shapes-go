package main

import (
	"fmt"
	"log"
	"math"
	"runtime"
	"strings"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"
)

const (
	windowWidth  = 1200
	windowHeight = 900
	windowTitle  = "OpenGL Cube, Octahedron, Icosahedron - Go"

	vertexShaderSource = `
		#version 410 core
		layout (location = 0) in vec3 aPos;
		layout (location = 1) in vec3 aColor;

		out vec3 VertexColor;

		uniform mat4 model;
		uniform mat4 view;
		uniform mat4 projection;

		void main()
		{
			gl_Position = projection * view * model * vec4(aPos, 1.0);
			VertexColor = aColor;
		}
	` + "\x00"

	fragmentShaderSource = `
		#version 410 core
		out vec4 FragColor;

		in vec3 VertexColor;

		uniform bool u_drawWireframe; // Flag to indicate wireframe rendering

		void main()
		{
			if (u_drawWireframe) {
				FragColor = vec4(0.0, 0.0, 0.0, 1.0); // Black color for wireframe
			} else {
				FragColor = vec4(VertexColor, 0.7f); // Vertex color for fill with alpha
			}
		}
	` + "\x00"
)

// Vertex represents a 3D point (position only).
type Vertex struct {
	X, Y, Z float32
}

// Shape holds OpenGL buffer information for a renderable object.
type Shape struct {
	VaoID        uint32
	ElementCount int32
	DrawMode     uint32 // e.g., gl.TRIANGLES
	UseElements  bool   // True if using EBO and gl.DrawElements
	// VBO and EBO IDs are implicitly managed by the VAO binding during setup
}

// --- Geometry Data ---

// Cube vertices (unique 8 vertices for indexed drawing)
var cubeUniqueVertices = []Vertex{
	{X: -0.5, Y: -0.5, Z: -0.5}, // 0 Bottom Left Back
	{X: 0.5, Y: -0.5, Z: -0.5},  // 1 Bottom Right Back
	{X: 0.5, Y: 0.5, Z: -0.5},   // 2 Top Right Back
	{X: -0.5, Y: 0.5, Z: -0.5},  // 3 Top Left Back
	{X: -0.5, Y: -0.5, Z: 0.5},  // 4 Bottom Left Front
	{X: 0.5, Y: -0.5, Z: 0.5},   // 5 Bottom Right Front
	{X: 0.5, Y: 0.5, Z: 0.5},    // 6 Top Right Front
	{X: -0.5, Y: 0.5, Z: 0.5},   // 7 Top Left Front
}

// Cube faces (12 triangles using indices for cubeUniqueVertices, CCW winding)
var cubeFaces = [][]int{
	// Back face
	{0, 3, 2}, {2, 1, 0},
	// Front face
	{4, 5, 6}, {6, 7, 4},
	// Left face
	{7, 3, 0}, {0, 4, 7},
	// Right face
	{5, 1, 2}, {2, 6, 5},
	// Bottom face
	{0, 1, 5}, {5, 4, 0},
	// Top face
	{3, 7, 6}, {6, 2, 3},
}

// Octahedron vertices (position 3f, color 3f) - 24 vertices
var octahedronVertices = []float32{
	// Define faces (vertices duplicated per triangle with face colors)
	// Face 1 (0, 2, 4) - Red - Indices refer to conceptual unique vertices: 0=R, 1=L, 2=T, 3=B, 4=F, 5=B
	0.6, 0.0, 0.0, 1.0, 0.0, 0.0, // v0 (Right)
	0.0, 0.6, 0.0, 1.0, 0.0, 0.0, // v2 (Top)
	0.0, 0.0, 0.6, 1.0, 0.0, 0.0, // v4 (Front)
	// Face 2 (0, 4, 3) - Green
	0.6, 0.0, 0.0, 0.0, 1.0, 0.0, // v0
	0.0, 0.0, 0.6, 0.0, 1.0, 0.0, // v4
	0.0, -0.6, 0.0, 0.0, 1.0, 0.0, // v3 (Bottom)
	// Face 3 (0, 3, 5) - Blue
	0.6, 0.0, 0.0, 0.0, 0.0, 1.0, // v0
	0.0, -0.6, 0.0, 0.0, 0.0, 1.0, // v3
	0.0, 0.0, -0.6, 0.0, 0.0, 1.0, // v5 (Back)
	// Face 4 (0, 5, 2) - Yellow
	0.6, 0.0, 0.0, 1.0, 1.0, 0.0, // v0
	0.0, 0.0, -0.6, 1.0, 1.0, 0.0, // v5
	0.0, 0.6, 0.0, 1.0, 1.0, 0.0, // v2
	// Face 5 (1, 2, 5) - Cyan
	-0.6, 0.0, 0.0, 0.0, 1.0, 1.0, // v1 (Left)
	0.0, 0.6, 0.0, 0.0, 1.0, 1.0, // v2
	0.0, 0.0, -0.6, 0.0, 1.0, 1.0, // v5
	// Face 6 (1, 5, 3) - Magenta
	-0.6, 0.0, 0.0, 1.0, 0.0, 1.0, // v1
	0.0, 0.0, -0.6, 1.0, 0.0, 1.0, // v5
	0.0, -0.6, 0.0, 1.0, 0.0, 1.0, // v3
	// Face 7 (1, 3, 4) - White
	-0.6, 0.0, 0.0, 1.0, 1.0, 1.0, // v1
	0.0, -0.6, 0.0, 1.0, 1.0, 1.0, // v3
	0.0, 0.0, 0.6, 1.0, 1.0, 1.0, // v4
	// Face 8 (1, 4, 2) - Orange
	-0.6, 0.0, 0.0, 1.0, 0.5, 0.0, // v1
	0.0, 0.0, 0.6, 1.0, 0.5, 0.0, // v4
	0.0, 0.6, 0.0, 1.0, 0.5, 0.0, // v2
}

// Icosahedron data
var (
	phiIco                 = (1.0 + float32(math.Sqrt(5.0))) / 2.0 // Use distinct name
	scaleFactorIco float32 = 0.4                                   // Keep scale factor, decreased size

	// Standard Icosahedron vertices using permutations of (0, ±1, ±phi)
	icoVertices = []Vertex{
		{X: -1 * scaleFactorIco, Y: phiIco * scaleFactorIco, Z: 0 * scaleFactorIco},  // 0
		{X: 1 * scaleFactorIco, Y: phiIco * scaleFactorIco, Z: 0 * scaleFactorIco},   // 1
		{X: -1 * scaleFactorIco, Y: -phiIco * scaleFactorIco, Z: 0 * scaleFactorIco}, // 2
		{X: 1 * scaleFactorIco, Y: -phiIco * scaleFactorIco, Z: 0 * scaleFactorIco},  // 3

		{X: 0 * scaleFactorIco, Y: -1 * scaleFactorIco, Z: phiIco * scaleFactorIco},  // 4
		{X: 0 * scaleFactorIco, Y: 1 * scaleFactorIco, Z: phiIco * scaleFactorIco},   // 5
		{X: 0 * scaleFactorIco, Y: -1 * scaleFactorIco, Z: -phiIco * scaleFactorIco}, // 6
		{X: 0 * scaleFactorIco, Y: 1 * scaleFactorIco, Z: -phiIco * scaleFactorIco},  // 7

		{X: phiIco * scaleFactorIco, Y: 0 * scaleFactorIco, Z: -1 * scaleFactorIco},  // 8
		{X: phiIco * scaleFactorIco, Y: 0 * scaleFactorIco, Z: 1 * scaleFactorIco},   // 9
		{X: -phiIco * scaleFactorIco, Y: 0 * scaleFactorIco, Z: -1 * scaleFactorIco}, // 10
		{X: -phiIco * scaleFactorIco, Y: 0 * scaleFactorIco, Z: 1 * scaleFactorIco},  // 11
	}

	// Standard Icosahedron faces (indices for vertices above, CCW winding)
	icoFaces = [][]int{
		{0, 11, 5}, {0, 5, 1}, {0, 1, 7}, {0, 7, 10}, {0, 10, 11}, // Faces around vertex 0
		{1, 5, 9}, {5, 11, 4}, {11, 10, 2}, {10, 7, 6}, {7, 1, 8}, // Adjacent faces
		{3, 9, 4}, {3, 4, 2}, {3, 2, 6}, {3, 6, 8}, {3, 8, 9}, // Faces around vertex 3
		{4, 9, 5}, {2, 4, 11}, {6, 2, 10}, {8, 6, 7}, {9, 8, 1}, // Remaining faces
	}
)

// Tetrahedron data
var (
	sqrt2                    = float32(math.Sqrt(2.0))
	sqrt6                    = float32(math.Sqrt(6.0))
	scaleFactorTetra float32 = 0.5 // Scale factor for tetrahedron

	// Tetrahedron vertices (centered at origin)
	tetraVertices = []Vertex{
		{X: 1.0 * scaleFactorTetra, Y: 0.0 * scaleFactorTetra, Z: -1.0 / sqrt2 * scaleFactorTetra},  // 0
		{X: -1.0 * scaleFactorTetra, Y: 0.0 * scaleFactorTetra, Z: -1.0 / sqrt2 * scaleFactorTetra}, // 1
		{X: 0.0 * scaleFactorTetra, Y: 1.0 * scaleFactorTetra, Z: 1.0 / sqrt2 * scaleFactorTetra},   // 2
		{X: 0.0 * scaleFactorTetra, Y: -1.0 * scaleFactorTetra, Z: 1.0 / sqrt2 * scaleFactorTetra},  // 3
	}

	// Tetrahedron faces (4 triangles, CCW winding)
	tetraFaces = [][]int{
		{0, 2, 3}, // Front face
		{1, 3, 2}, // Left face (? Check winding)
		{0, 1, 2}, // Right face (? Check winding)
		{0, 3, 1}, // Bottom face (? Check winding)
		// Let's re-verify CCW winding based on vertices
		// Face 0: (0, 2, 3) - OK
		// Face 1: (1, 3, 2) -> (1, 2, 3) for CCW
		// Face 2: (0, 1, 2) -> (0, 2, 1) ?? No, (0,1,2) seems correct for right face.
		// Face 3: (0, 3, 1) -> (0, 1, 3) for CCW
		// Corrected faces:
		{0, 2, 3},
		{1, 2, 3}, // Corrected
		{0, 1, 2}, // Corrected
		{0, 1, 3}, // Corrected
	}
)

// Dodecahedron data (adapted from Stack Overflow example)
var (
	phiDodeca                 = (1.0 + float32(math.Sqrt(5.0))) / 2.0 // Golden ratio
	invPhiDodeca              = 1.0 / phiDodeca                       // 1 / phi
	scaleFactorDodeca float32 = 0.35                                  // Slightly larger scale for visibility (restored scale)

	// Vertices based on Stack Overflow example (using phi and 1/phi)
	dodecaVertices = []Vertex{
		{X: 1 * scaleFactorDodeca, Y: 1 * scaleFactorDodeca, Z: 1 * scaleFactorDodeca},                      // 0
		{X: 1 * scaleFactorDodeca, Y: 1 * scaleFactorDodeca, Z: -1 * scaleFactorDodeca},                     // 1
		{X: 1 * scaleFactorDodeca, Y: -1 * scaleFactorDodeca, Z: 1 * scaleFactorDodeca},                     // 2
		{X: 1 * scaleFactorDodeca, Y: -1 * scaleFactorDodeca, Z: -1 * scaleFactorDodeca},                    // 3
		{X: -1 * scaleFactorDodeca, Y: 1 * scaleFactorDodeca, Z: 1 * scaleFactorDodeca},                     // 4
		{X: -1 * scaleFactorDodeca, Y: 1 * scaleFactorDodeca, Z: -1 * scaleFactorDodeca},                    // 5
		{X: -1 * scaleFactorDodeca, Y: -1 * scaleFactorDodeca, Z: 1 * scaleFactorDodeca},                    // 6
		{X: -1 * scaleFactorDodeca, Y: -1 * scaleFactorDodeca, Z: -1 * scaleFactorDodeca},                   // 7
		{X: 0 * scaleFactorDodeca, Y: invPhiDodeca * scaleFactorDodeca, Z: phiDodeca * scaleFactorDodeca},   // 8
		{X: 0 * scaleFactorDodeca, Y: invPhiDodeca * scaleFactorDodeca, Z: -phiDodeca * scaleFactorDodeca},  // 9
		{X: 0 * scaleFactorDodeca, Y: -invPhiDodeca * scaleFactorDodeca, Z: phiDodeca * scaleFactorDodeca},  // 10
		{X: 0 * scaleFactorDodeca, Y: -invPhiDodeca * scaleFactorDodeca, Z: -phiDodeca * scaleFactorDodeca}, // 11
		{X: invPhiDodeca * scaleFactorDodeca, Y: phiDodeca * scaleFactorDodeca, Z: 0 * scaleFactorDodeca},   // 12
		{X: invPhiDodeca * scaleFactorDodeca, Y: -phiDodeca * scaleFactorDodeca, Z: 0 * scaleFactorDodeca},  // 13
		{X: -invPhiDodeca * scaleFactorDodeca, Y: phiDodeca * scaleFactorDodeca, Z: 0 * scaleFactorDodeca},  // 14
		{X: -invPhiDodeca * scaleFactorDodeca, Y: -phiDodeca * scaleFactorDodeca, Z: 0 * scaleFactorDodeca}, // 15
		{X: phiDodeca * scaleFactorDodeca, Y: 0 * scaleFactorDodeca, Z: invPhiDodeca * scaleFactorDodeca},   // 16
		{X: phiDodeca * scaleFactorDodeca, Y: 0 * scaleFactorDodeca, Z: -invPhiDodeca * scaleFactorDodeca},  // 17
		{X: -phiDodeca * scaleFactorDodeca, Y: 0 * scaleFactorDodeca, Z: invPhiDodeca * scaleFactorDodeca},  // 18
		{X: -phiDodeca * scaleFactorDodeca, Y: 0 * scaleFactorDodeca, Z: -invPhiDodeca * scaleFactorDodeca}, // 19
	}

	// Faces based on Stack Overflow example (Original order)
	dodecaFaces = [][]int{
		// Final check against the SO `faces` array directly
		{0, 16, 2, 10, 8},
		{0, 8, 4, 14, 12},
		{16, 17, 1, 12, 0},
		{1, 9, 11, 3, 17},
		{1, 12, 14, 5, 9},
		{2, 13, 15, 6, 10},
		{13, 3, 17, 16, 2},
		{3, 11, 7, 15, 13},
		{4, 8, 10, 6, 18},
		{14, 5, 19, 18, 4},
		{5, 19, 7, 11, 9},
		{15, 7, 19, 18, 6},
	}
)

// Generates triangle indices for use with gl.DrawElements.
// Handles triangles directly and uses fan triangulation for polygons.
func generateTriangleIndices(faces [][]int) []uint32 {
	var indices []uint32
	for _, face := range faces {
		if len(face) == 3 {
			// Direct triangle
			indices = append(indices, uint32(face[0]), uint32(face[1]), uint32(face[2]))
		} else if len(face) > 3 {
			// Fan triangulation: 0,1,2 / 0,2,3 / 0,3,4 / ...
			for i := 1; i < len(face)-1; i++ {
				indices = append(indices, uint32(face[0]), uint32(face[i]), uint32(face[i+1]))
			}
		} // Ignore faces with < 3 vertices
	}
	return indices
}

// Generates a flat []float32 array with position and calculated color data for unique vertices.
func generateUniqueDrawableVertices(vertices []Vertex, baseScale float32) []float32 {
	var drawableVerts = make([]float32, 0, len(vertices)*6) // 6 floats per vertex (3 pos, 3 color)
	for _, v := range vertices {
		// Position
		drawableVerts = append(drawableVerts, v.X, v.Y, v.Z)
		// Procedural Color based on position (normalized to 0.0-1.0 range approx)
		r := (v.X/baseScale)*0.5 + 0.5
		g := (v.Y/baseScale)*0.5 + 0.5
		b := (v.Z/baseScale)*0.5 + 0.5
		drawableVerts = append(drawableVerts, r, g, b)
	}
	return drawableVerts
}

// setupBuffersArrays configures VAO/VBO for drawing with gl.DrawArrays.
func setupBuffersArrays(vertices []float32) Shape {
	var vbo uint32
	gl.GenBuffers(1, &vbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	gl.BufferData(gl.ARRAY_BUFFER, len(vertices)*4, gl.Ptr(vertices), gl.STATIC_DRAW)

	var vao uint32
	gl.GenVertexArrays(1, &vao)
	gl.BindVertexArray(vao)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo) // Bind VBO to this VAO

	// Position attribute
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, 6*4, gl.PtrOffset(0))
	gl.EnableVertexAttribArray(0)
	// Color attribute
	gl.VertexAttribPointer(1, 3, gl.FLOAT, false, 6*4, gl.PtrOffset(3*4))
	gl.EnableVertexAttribArray(1)

	gl.BindVertexArray(0)             // Unbind VAO
	gl.BindBuffer(gl.ARRAY_BUFFER, 0) // Unbind VBO

	return Shape{
		VaoID:        vao,
		ElementCount: int32(len(vertices) / 6), // 6 floats per vertex
		DrawMode:     gl.TRIANGLES,
		UseElements:  false,
	}
}

// setupBuffersElements configures VAO/VBO/EBO for drawing with gl.DrawElements.
func setupBuffersElements(vertices []float32, indices []uint32) Shape {
	var vbo uint32
	gl.GenBuffers(1, &vbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	gl.BufferData(gl.ARRAY_BUFFER, len(vertices)*4, gl.Ptr(vertices), gl.STATIC_DRAW)

	var ebo uint32
	gl.GenBuffers(1, &ebo)
	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, ebo)
	gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, len(indices)*4, gl.Ptr(indices), gl.STATIC_DRAW)

	var vao uint32
	gl.GenVertexArrays(1, &vao)
	gl.BindVertexArray(vao)

	gl.BindBuffer(gl.ARRAY_BUFFER, vbo) // Bind VBO to this VAO
	// Position attribute
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, 6*4, gl.PtrOffset(0))
	gl.EnableVertexAttribArray(0)
	// Color attribute
	gl.VertexAttribPointer(1, 3, gl.FLOAT, false, 6*4, gl.PtrOffset(3*4))
	gl.EnableVertexAttribArray(1)

	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, ebo) // Bind EBO to this VAO

	gl.BindVertexArray(0) // Unbind VAO
	// Important: Unbind EBO *after* unbinding VAO
	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, 0)
	gl.BindBuffer(gl.ARRAY_BUFFER, 0) // Unbind VBO

	return Shape{
		VaoID:        vao,
		ElementCount: int32(len(indices)),
		DrawMode:     gl.TRIANGLES,
		UseElements:  true,
	}
}

// drawShape handles drawing a shape (fill and wireframe).
func drawShape(shape Shape, program uint32, model mgl32.Mat4, modelLoc int32, wireframeLoc int32) {
	gl.UniformMatrix4fv(modelLoc, 1, false, &model[0])
	gl.BindVertexArray(shape.VaoID)

	// For transparency: disable depth writing, but keep depth testing
	gl.DepthMask(false)

	// Draw Fill
	gl.Enable(gl.POLYGON_OFFSET_FILL)
	gl.PolygonMode(gl.FRONT_AND_BACK, gl.FILL)
	gl.PolygonOffset(1.0, 1.0)
	gl.Uniform1i(wireframeLoc, 0) // Set shader for fill color
	if shape.UseElements {
		gl.DrawElements(shape.DrawMode, shape.ElementCount, gl.UNSIGNED_INT, nil)
	} else {
		gl.DrawArrays(shape.DrawMode, 0, shape.ElementCount)
	}
	gl.Disable(gl.POLYGON_OFFSET_FILL)

	// Re-enable depth writing for the opaque wireframe
	gl.DepthMask(true)

	// Draw Wireframe
	gl.Enable(gl.POLYGON_OFFSET_LINE)
	gl.PolygonMode(gl.FRONT_AND_BACK, gl.LINE)
	gl.PolygonOffset(-1.0, -1.0)
	gl.LineWidth(2.0)
	gl.Uniform1i(wireframeLoc, 1) // Set shader for wireframe color
	if shape.UseElements {
		gl.DrawElements(shape.DrawMode, shape.ElementCount, gl.UNSIGNED_INT, nil)
	} else {
		gl.DrawArrays(shape.DrawMode, 0, shape.ElementCount)
	}
	gl.LineWidth(1.0)
	gl.Disable(gl.POLYGON_OFFSET_LINE)

	gl.BindVertexArray(0)
}

func main() {
	runtime.LockOSThread()

	if err := glfw.Init(); err != nil {
		log.Fatalln("failed to initialize glfw:", err)
	}
	defer glfw.Terminate()

	glfw.WindowHint(glfw.Resizable, glfw.False)
	glfw.WindowHint(glfw.ContextVersionMajor, 4)
	glfw.WindowHint(glfw.ContextVersionMinor, 1)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)

	window, err := glfw.CreateWindow(windowWidth, windowHeight, windowTitle, nil, nil)
	if err != nil {
		log.Fatalln("failed to create glfw window:", err)
	}
	window.MakeContextCurrent()

	if err := gl.Init(); err != nil {
		log.Fatalln("failed to initialize gl:", err)
	}

	version := gl.GoStr(gl.GetString(gl.VERSION))
	log.Println("OpenGL version", version)

	// Build and compile our shader program
	program, err := newProgram(vertexShaderSource, fragmentShaderSource)
	if err != nil {
		log.Fatalln("Failed to create shader program:", err)
	}
	gl.UseProgram(program) // Activate shader program once

	// --- Setup Shapes ---
	// Cube (Indexed Drawing)
	cubeDrawableVertices := generateUniqueDrawableVertices(cubeUniqueVertices, 0.5) // Use 0.5 as base scale for cube color
	cubeTriangleIndices := generateTriangleIndices(cubeFaces)
	cubeShape := setupBuffersElements(cubeDrawableVertices, cubeTriangleIndices)

	// Octahedron (Array Drawing - kept for diversity)
	octaShape := setupBuffersArrays(octahedronVertices)

	// Icosahedron (Indexed Drawing)
	icoDrawableVertices := generateUniqueDrawableVertices(icoVertices, scaleFactorIco)
	icoTriangleIndices := generateTriangleIndices(icoFaces)
	icoShape := setupBuffersElements(icoDrawableVertices, icoTriangleIndices)

	// Tetrahedron (Indexed Drawing)
	tetraDrawableVertices := generateUniqueDrawableVertices(tetraVertices, scaleFactorTetra)
	tetraTriangleIndices := generateTriangleIndices(tetraFaces)
	tetraShape := setupBuffersElements(tetraDrawableVertices, tetraTriangleIndices)

	// Dodecahedron data and setup removed due to rendering issues
	/*
		// Dodecahedron (Indexed Drawing - DEBUG: Draw only the first face)
		dodecaDrawableVertices := generateUniqueDrawableVertices(dodecaVertices, scaleFactorDodeca)
		// dodecaTriangleIndices := generateTriangleIndices(dodecaFaces)
		dodecaTriangleIndices := generateTriangleIndices(dodecaFaces[0:1]) // Use only the first face
		dodecaShape := setupBuffersElements(dodecaDrawableVertices, dodecaTriangleIndices)
	*/

	// Configure global settings
	gl.Enable(gl.DEPTH_TEST)
	gl.DepthFunc(gl.LESS)
	gl.ClearColor(0.7, 0.7, 0.7, 1.0) // Lighter grey background
	gl.Disable(gl.CULL_FACE)          // Disable again to see all triangles

	// Enable blending for transparency
	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)

	// Enable polygon offset (used in drawShape)
	// Note: We enable/disable specific offsets (FILL/LINE) within drawShape
	gl.Enable(gl.POLYGON_OFFSET_FILL)
	gl.Enable(gl.POLYGON_OFFSET_LINE)

	// Uniform locations (get them once)
	modelLoc := gl.GetUniformLocation(program, gl.Str("model"+"\x00"))
	viewLoc := gl.GetUniformLocation(program, gl.Str("view"+"\x00"))
	projLoc := gl.GetUniformLocation(program, gl.Str("projection"+"\x00"))
	wireframeLoc := gl.GetUniformLocation(program, gl.Str("u_drawWireframe"+"\x00"))

	// --- MVP Matrices (Set Projection and View once) ---
	projection := mgl32.Perspective(mgl32.DegToRad(45.0), float32(windowWidth)/float32(windowHeight), 0.1, 100.0)
	view := mgl32.LookAtV(mgl32.Vec3{0, 0, 4}, mgl32.Vec3{0, 0, 0}, mgl32.Vec3{0, 1, 0}) // Move camera back slightly
	gl.UniformMatrix4fv(projLoc, 1, false, &projection[0])
	gl.UniformMatrix4fv(viewLoc, 1, false, &view[0])

	// --- Animation Variables ---
	angleCube := float32(0.0)
	angleOcta := float32(math.Pi / 4.0)
	angleIco := float32(math.Pi / 2.0)
	angleTetra := float32(0.0) // Start tetra angle
	// angleDodeca := float32(math.Pi * 3.0 / 4.0) // Remove dodecahedron angle
	previousTime := glfw.GetTime()

	// Define rotation axes
	rotationAxisCube := mgl32.Vec3{0.5, 1.0, 0.0}.Normalize()
	rotationAxisOcta := mgl32.Vec3{0.0, 1.0, 0.5}.Normalize()
	rotationAxisIco := mgl32.Vec3{1.0, 0.0, 0.5}.Normalize()
	rotationAxisTetra := mgl32.Vec3{0.0, 1.0, 0.0}.Normalize() // Rotate around Y axis
	// rotationAxisDodeca := mgl32.Vec3{0.5, 0.0, 1.0}.Normalize() // Remove dodecahedron axis

	// Main loop
	for !window.ShouldClose() {
		currentTime := glfw.GetTime()
		deltaTime := float32(currentTime - previousTime)
		previousTime = currentTime

		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

		// Note: gl.UseProgram(program) is already called outside the loop

		// --- Update and Draw Cube ---
		angleCube += mgl32.DegToRad(50.0) * deltaTime
		modelCube := mgl32.Translate3D(-1.3, 0.7, 0).Mul4( // Position cube top-left
			mgl32.HomogRotate3D(angleCube, rotationAxisCube),
		)
		drawShape(cubeShape, program, modelCube, modelLoc, wireframeLoc)

		// --- Update and Draw Tetrahedron ---
		angleTetra += mgl32.DegToRad(80.0) * deltaTime     // Rotation speed for tetra
		modelTetra := mgl32.Translate3D(1.3, 0.7, 0).Mul4( // Position tetrahedron top-right
			mgl32.HomogRotate3D(angleTetra, rotationAxisTetra),
		)
		drawShape(tetraShape, program, modelTetra, modelLoc, wireframeLoc)

		// --- Update and Draw Icosahedron ---
		angleIco += mgl32.DegToRad(90.0) * deltaTime
		modelIco := mgl32.Translate3D(-1.3, -0.7, 0).Mul4( // Position icosahedron bottom-left
			mgl32.HomogRotate3D(angleIco, rotationAxisIco),
		)
		drawShape(icoShape, program, modelIco, modelLoc, wireframeLoc)

		// --- Update and Draw Octahedron ---
		angleOcta += mgl32.DegToRad(70.0) * deltaTime
		modelOcta := mgl32.Translate3D(1.3, -0.7, 0).Mul4( // Position octahedron bottom-right
			mgl32.HomogRotate3D(angleOcta, rotationAxisOcta),
		)
		drawShape(octaShape, program, modelOcta, modelLoc, wireframeLoc)

		// Reset polygon mode to default (optional, good practice)
		gl.PolygonMode(gl.FRONT_AND_BACK, gl.FILL)

		// Maintenance
		window.SwapBuffers()
		glfw.PollEvents()
	}
}

// newProgram compiles and links shaders into a program.
func newProgram(vertexShaderSource, fragmentShaderSource string) (uint32, error) {
	vertexShader, err := compileShader(vertexShaderSource, gl.VERTEX_SHADER)
	if err != nil {
		return 0, err
	}

	fragmentShader, err := compileShader(fragmentShaderSource, gl.FRAGMENT_SHADER)
	if err != nil {
		return 0, err
	}

	program := gl.CreateProgram()

	gl.AttachShader(program, vertexShader)
	gl.AttachShader(program, fragmentShader)
	gl.LinkProgram(program)

	var status int32
	gl.GetProgramiv(program, gl.LINK_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetProgramiv(program, gl.INFO_LOG_LENGTH, &logLength)

		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetProgramInfoLog(program, logLength, nil, gl.Str(log))

		return 0, fmt.Errorf("failed to link program: %v", log)
	}

	gl.DeleteShader(vertexShader)
	gl.DeleteShader(fragmentShader)

	return program, nil
}

// compileShader compiles a single shader.
func compileShader(source string, shaderType uint32) (uint32, error) {
	shader := gl.CreateShader(shaderType)

	csources, free := gl.Strs(source)
	gl.ShaderSource(shader, 1, csources, nil)
	free()
	gl.CompileShader(shader)

	var status int32
	gl.GetShaderiv(shader, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetShaderiv(shader, gl.INFO_LOG_LENGTH, &logLength)

		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetShaderInfoLog(shader, logLength, nil, gl.Str(log))

		return 0, fmt.Errorf("failed to compile %v shader: %v", shaderType, log)
	}

	return shader, nil
}
