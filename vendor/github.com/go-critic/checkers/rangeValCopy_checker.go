package checkers

import (
	"go/ast"

	"github.com/go-lintpack/lintpack"
	"github.com/go-lintpack/lintpack/astwalk"
)

func init() {
	var info lintpack.CheckerInfo
	info.Name = "rangeValCopy"
	info.Tags = []string{"performance"}
	info.Summary = "Detects loops that copy big objects during each iteration"
	info.Details = "Suggests to use index access or take address and make use pointer instead."
	info.Before = `
xs := make([][1024]byte, length)
for _, x := range xs {
	// Loop body.
}`
	info.After = `
xs := make([][1024]byte, length)
for i := range xs {
	x := &xs[i]
	// Loop body.
}`

	lintpack.AddChecker(&info, func(ctx *lintpack.CheckerContext) lintpack.FileWalker {
		c := &rangeValCopyChecker{ctx: ctx}
		c.sizeThreshold = int64(c.ctx.Params.Int("sizeThreshold", 128))
		c.skipTestFuncs = c.ctx.Params.Bool("skipTestFuncs", true)
		return astwalk.WalkerForStmt(c)
	})
}

type rangeValCopyChecker struct {
	astwalk.WalkHandler
	ctx *lintpack.CheckerContext

	sizeThreshold int64
	skipTestFuncs bool
}

func (c *rangeValCopyChecker) EnterFunc(fn *ast.FuncDecl) bool {
	return fn.Body != nil &&
		!(c.skipTestFuncs && isUnitTestFunc(c.ctx, fn))
}

func (c *rangeValCopyChecker) VisitStmt(stmt ast.Stmt) {
	rng, ok := stmt.(*ast.RangeStmt)
	if !ok || rng.Value == nil {
		return
	}
	typ := c.ctx.TypesInfo.TypeOf(rng.Value)
	if typ == nil {
		return
	}
	if size := c.ctx.SizesInfo.Sizeof(typ); size >= c.sizeThreshold {
		c.warn(rng, size)
	}
}

func (c *rangeValCopyChecker) warn(node ast.Node, size int64) {
	c.ctx.Warn(node, "each iteration copies %d bytes (consider pointers or indexing)", size)
}
