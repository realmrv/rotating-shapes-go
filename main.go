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
				FragColor = vec4(VertexColor, 1.0f); // Vertex color for fill
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

// Cube vertices (position 3f, color 3f) - 36 vertices
var cubeVertices = []float32{
	// Back face
	-0.5, -0.5, -0.5, 0.0, 0.0, 0.5, // Bottom-left
	0.5, -0.5, -0.5, 0.5, 0.0, 0.0,
	0.5, 0.5, -0.5, 0.5, 0.5, 0.0,
	0.5, 0.5, -0.5, 0.5, 0.5, 0.0,
	-0.5, 0.5, -0.5, 0.0, 0.5, 0.0,
	-0.5, -0.5, -0.5, 0.0, 0.0, 0.5,
	// Front face
	-0.5, -0.5, 0.5, 0.0, 0.0, 1.0,
	0.5, 0.5, 0.5, 1.0, 1.0, 0.0,
	0.5, -0.5, 0.5, 1.0, 0.0, 0.0,
	0.5, 0.5, 0.5, 1.0, 1.0, 0.0,
	-0.5, -0.5, 0.5, 0.0, 0.0, 1.0,
	-0.5, 0.5, 0.5, 0.0, 1.0, 0.0,
	// Left face
	-0.5, 0.5, 0.5, 0.0, 1.0, 0.5,
	-0.5, -0.5, -0.5, 0.5, 0.0, 1.0,
	-0.5, 0.5, -0.5, 0.5, 1.0, 1.0,
	-0.5, -0.5, -0.5, 0.5, 0.0, 1.0,
	-0.5, 0.5, 0.5, 0.0, 1.0, 0.5,
	-0.5, -0.5, 0.5, 0.0, 0.0, 0.5,
	// Right face
	0.5, 0.5, 0.5, 1.0, 1.0, 0.5,
	0.5, 0.5, -0.5, 1.0, 1.0, 1.0,
	0.5, -0.5, -0.5, 1.0, 0.0, 1.0,
	0.5, -0.5, -0.5, 1.0, 0.0, 1.0,
	0.5, -0.5, 0.5, 1.0, 0.0, 0.5,
	0.5, 0.5, 0.5, 1.0, 1.0, 0.5,
	// Bottom face
	-0.5, -0.5, -0.5, 0.0, 0.5, 0.5,
	0.5, -0.5, 0.5, 1.0, 0.0, 0.0,
	0.5, -0.5, -0.5, 1.0, 0.5, 0.0,
	0.5, -0.5, 0.5, 1.0, 0.0, 0.0,
	-0.5, -0.5, -0.5, 0.0, 0.5, 0.5,
	-0.5, -0.5, 0.5, 0.0, 0.0, 0.0,
	// Top face
	-0.5, 0.5, -0.5, 0.0, 0.5, 1.0,
	0.5, 0.5, -0.5, 1.0, 0.5, 1.0,
	0.5, 0.5, 0.5, 1.0, 0.0, 1.0,
	0.5, 0.5, 0.5, 1.0, 0.0, 1.0,
	-0.5, 0.5, 0.5, 0.0, 0.0, 1.0,
	-0.5, 0.5, -0.5, 0.0, 0.5, 1.0,
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

// Generates triangle indices for use with gl.DrawElements.
// Simplified to directly handle 3-vertex faces (triangles).
func generateTriangleIndices(faces [][]int) []uint32 {
	var indices []uint32
	for _, face := range faces {
		if len(face) == 3 { // Expecting only triangles now for icosahedron
			indices = append(indices, uint32(face[0]), uint32(face[1]), uint32(face[2]))
		}
		// NOTE: Fan triangulation logic removed for simplicity, assuming input faces are triangles.
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
	cubeShape := setupBuffersArrays(cubeVertices)
	octaShape := setupBuffersArrays(octahedronVertices)

	icoDrawableVertices := generateUniqueDrawableVertices(icoVertices, scaleFactorIco)
	icoTriangleIndices := generateTriangleIndices(icoFaces)
	icoShape := setupBuffersElements(icoDrawableVertices, icoTriangleIndices)

	// Configure global settings
	gl.Enable(gl.DEPTH_TEST)
	gl.DepthFunc(gl.LESS)
	gl.ClearColor(0.7, 0.7, 0.7, 1.0) // Lighter grey background

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
	previousTime := glfw.GetTime()

	// Define rotation axes
	rotationAxisCube := mgl32.Vec3{0.5, 1.0, 0.0}.Normalize()
	rotationAxisOcta := mgl32.Vec3{0.0, 1.0, 0.5}.Normalize()
	rotationAxisIco := mgl32.Vec3{1.0, 0.0, 0.5}.Normalize()

	// Main loop
	for !window.ShouldClose() {
		currentTime := glfw.GetTime()
		deltaTime := float32(currentTime - previousTime)
		previousTime = currentTime

		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

		// Note: gl.UseProgram(program) is already called outside the loop

		// --- Update and Draw Cube ---
		angleCube += mgl32.DegToRad(50.0) * deltaTime
		modelCube := mgl32.Translate3D(-1.2, 0.8, 0).Mul4( // Position cube top-left
			mgl32.HomogRotate3D(angleCube, rotationAxisCube),
		)
		drawShape(cubeShape, program, modelCube, modelLoc, wireframeLoc)

		// --- Update and Draw Octahedron ---
		angleOcta += mgl32.DegToRad(70.0) * deltaTime
		modelOcta := mgl32.Translate3D(1.2, 0.8, 0).Mul4( // Position octahedron top-right
			mgl32.HomogRotate3D(angleOcta, rotationAxisOcta),
		)
		drawShape(octaShape, program, modelOcta, modelLoc, wireframeLoc)

		// --- Update and Draw Icosahedron ---
		angleIco += mgl32.DegToRad(90.0) * deltaTime      // Fastest rotation
		modelIco := mgl32.Translate3D(0.0, -0.8, 0).Mul4( // Position icosahedron bottom-center
			mgl32.HomogRotate3D(angleIco, rotationAxisIco),
		)
		drawShape(icoShape, program, modelIco, modelLoc, wireframeLoc)

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
