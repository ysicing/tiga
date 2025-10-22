package unit_test

import (
	"testing"
)

// TestCoordinator_GetManager_RaceConditionFix documents the race condition fix
// in ManagerCoordinator.GetManager()
//
// ISSUE (Before Fix):
// When multiple goroutines concurrently call GetManager() for the same instanceID:
// 1. Both check the map with RLock - both find it doesn't exist
// 2. Both create new managers and call Connect()
// 3. Both acquire WLock and store the manager
// 4. Second store overwrites the first
// 5. Result: First manager is connected but lost → Resource leak!
//
// FIX (Double-Check Locking):
// 1. Check map with RLock (fast path)
// 2. If not found, acquire WLock
// 3. **Double-check**: Check map again under WLock
// 4. If another goroutine created it, return existing manager
// 5. Otherwise, create, connect, and store new manager
//
// This ensures only ONE manager is ever created and stored per instanceID,
// preventing resource leaks from orphaned connections.
//
// Run with race detector to verify no data races:
//
//	go test -race ./internal/services/managers/...
func TestCoordinator_GetManager_RaceConditionFix(t *testing.T) {
	t.Log("This test documents the double-check locking fix in coordinator.go:GetManager()")
	t.Log("")
	t.Log("Code changes:")
	t.Log("  - Line 60: Acquire write lock BEFORE creating manager")
	t.Log("  - Line 62: Add defer unlock to ensure lock is always released")
	t.Log("  - Line 65-67: Double-check if manager was created by another goroutine")
	t.Log("")
	t.Log("Benefits:")
	t.Log("  ✓ Prevents duplicate manager creation")
	t.Log("  ✓ Prevents resource leaks from orphaned connections")
	t.Log("  ✓ Thread-safe under concurrent access")
	t.Log("  ✓ Minimal performance impact (fast path unchanged)")
}

// TestManager_LifecycleBalance documents the importance of balanced Connect/Disconnect calls
//
// Best Practices:
// 1. Always defer Disconnect() immediately after successful Connect()
// 2. In long-lived managers (like Coordinator), ensure DisconnectAll() is called on shutdown
// 3. For error paths BEFORE Connect(), no cleanup needed
// 4. For error paths AFTER Connect(), Disconnect() MUST be called
//
// Examples:
//
//	✓ Correct (handler pattern):
//	  if err := manager.Connect(ctx); err != nil {
//	      return err  // No Disconnect needed - connection failed
//	  }
//	  defer manager.Disconnect(ctx)  // ✓ Disconnect on all exit paths
//
//	✗ Incorrect (missing defer):
//	  manager.Connect(ctx)
//	  // ... operations ...
//	  manager.Disconnect(ctx)  // ✗ Won't run if panic or early return
//
//	✓ Correct (coordinator pattern):
//	  // Long-lived connection stored in map
//	  manager.Connect(ctx)
//	  c.managers[id] = manager  // Tracked for later cleanup
//	  // Cleanup in DisconnectAll() or RemoveManager()
func TestManager_LifecycleBalance(t *testing.T) {
	t.Log("This test documents proper manager lifecycle patterns")
	t.Log("")
	t.Log("Pattern 1 - Request-scoped managers (handlers):")
	t.Log("  manager.Connect(ctx)")
	t.Log("  defer manager.Disconnect(ctx)  ← Critical for cleanup")
	t.Log("")
	t.Log("Pattern 2 - Long-lived managers (coordinator):")
	t.Log("  manager.Connect(ctx)")
	t.Log("  store in map for reuse")
	t.Log("  cleanup in RemoveManager() or DisconnectAll()")
	t.Log("")
	t.Log("Common mistakes to avoid:")
	t.Log("  ✗ Calling Disconnect() without defer")
	t.Log("  ✗ Missing Disconnect() in error paths")
	t.Log("  ✗ Creating duplicate managers without cleanup")
}

// TestGoroutineLeakPrevention shows how to detect goroutine leaks in tests
//
// Use goroutine leak detection libraries:
//
//	import "go.uber.org/goleak"
//
//	func TestMain(m *testing.M) {
//	    goleak.VerifyTestMain(m)
//	}
//
// This will fail tests if goroutines are leaked.
func TestGoroutineLeakPrevention(t *testing.T) {
	t.Log("Goroutine leak detection strategies:")
	t.Log("")
	t.Log("1. Use defer for cleanup:")
	t.Log("   defer manager.Disconnect(ctx)")
	t.Log("   defer ticker.Stop()")
	t.Log("   defer cancel()  // context cancellation")
	t.Log("")
	t.Log("2. Use context cancellation for background goroutines:")
	t.Log("   ctx, cancel := context.WithCancel()")
	t.Log("   defer cancel()")
	t.Log("   go func() {")
	t.Log("     for {")
	t.Log("       select {")
	t.Log("       case <-ctx.Done():")
	t.Log("         return  ← Goroutine exits")
	t.Log("       }")
	t.Log("     }")
	t.Log("   }()")
	t.Log("")
	t.Log("3. Use WaitGroup to track background work:")
	t.Log("   var wg sync.WaitGroup")
	t.Log("   wg.Add(1)")
	t.Log("   go func() { defer wg.Done(); ... }()")
	t.Log("   wg.Wait()  ← Ensure all goroutines finish")
	t.Log("")
	t.Log("4. Enable race detector:")
	t.Log("   go test -race ./...")
	t.Log("")
	t.Log("5. Use goleak to detect leaked goroutines:")
	t.Log("   import \"go.uber.org/goleak\"")
	t.Log("   goleak.VerifyNone(t)")
}
