# rotating-shapes

A simple 3D rotating shapes demo using Go.

## Building and Running

To build the project:

```bash
go build
```

To run the project:

```bash
./rotating-shapes 
```

## Dependencies

- Go 1.24.1 or later
- [github.com/go-gl/gl](https://github.com/go-gl/gl)
- [github.com/go-gl/glfw/v3.3/glfw](https://github.com/go-gl/glfw)
- [github.com/go-gl/mathgl](https://github.com/go-gl/mathgl)

## Features

- Displays four rotating 3D shapes: Cube, Octahedron, Icosahedron, and Tetrahedron.
- Each shape rotates independently around its own axis at different speeds.
- Renders the scene using OpenGL (v4.1 Core Profile).
- Utilizes vertex and fragment shaders for rendering.
- Implements blending for transparency effects.
- Renders both the filled shape and its wireframe using polygon offset to avoid z-fighting.
- Vertex colors are procedurally generated based on their position.
