package storage

import (
	"testing"
)

// openTestDB creates an in-memory SQLite database and runs all migrations,
// giving each test a clean schema.
func openTestDB(t *testing.T) *Database {
	t.Helper()
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("openTestDB: %v", err)
	}
	if err := db.Migrate(); err != nil {
		t.Fatalf("openTestDB Migrate: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Logf("failed to close database: %v", err)
		}
	})
	return db
}

// addGroup adds a group and returns its autoincrement ID via All().
func addGroup(t *testing.T, dgs *DriverGroupStorage, group DriverGroup) uint {
	t.Helper()
	if err := dgs.Add(group); err != nil {
		t.Fatalf("Add: %v", err)
	}
	all, err := dgs.All()
	if err != nil {
		t.Fatalf("All after Add: %v", err)
	}
	return all[len(all)-1].Id
}

// ==================== DriverGroupStorage CRUD ====================

func TestDriverGroupStorage_All_Empty(t *testing.T) {
	db := openTestDB(t)
	dgs := NewDriverGroupStorage(db)

	groups, err := dgs.All()
	if err != nil {
		t.Fatalf("All: %v", err)
	}
	if len(groups) != 0 {
		t.Errorf("expected 0 groups, got %d", len(groups))
	}
}

func TestDriverGroupStorage_Get_NotFound(t *testing.T) {
	db := openTestDB(t)
	dgs := NewDriverGroupStorage(db)

	if _, err := dgs.Get(9999); err == nil {
		t.Fatal("expected error for nonexistent group, got nil")
	}
}

func TestDriverGroupStorage_Add_And_All(t *testing.T) {
	db := openTestDB(t)
	dgs := NewDriverGroupStorage(db)

	id1 := addGroup(t, dgs, DriverGroup{Name: "G1", Type: Network})
	id2 := addGroup(t, dgs, DriverGroup{Name: "G2", Type: Display})

	if id1 == 0 || id2 == 0 {
		t.Fatal("expected non-zero IDs from autoincrement")
	}

	all, err := dgs.All()
	if err != nil {
		t.Fatalf("All: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(all))
	}
	if all[0].Id != id1 || all[1].Id != id2 {
		t.Errorf("unexpected order: [%d, %d]", all[0].Id, all[1].Id)
	}
}

func TestDriverGroupStorage_Add_AssignsIdsToDrivers(t *testing.T) {
	db := openTestDB(t)
	dgs := NewDriverGroupStorage(db)

	id := addGroup(t, dgs, DriverGroup{
		Name:    "Group with drivers",
		Type:    Network,
		Drivers: []*Driver{{Name: "D1"}, {Name: "D2"}},
	})

	group, err := dgs.Get(id)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(group.Drivers) != 2 {
		t.Fatalf("expected 2 drivers, got %d", len(group.Drivers))
	}

	seen := make(map[uint]bool)
	for _, d := range group.Drivers {
		if d.Id == 0 {
			t.Errorf("driver %q should have a non-zero ID from autoincrement", d.Name)
		}
		if seen[d.Id] {
			t.Errorf("duplicate driver ID: %d", d.Id)
		}
		seen[d.Id] = true
	}
}

func TestDriverGroupStorage_Update_ScalarFields(t *testing.T) {
	db := openTestDB(t)
	dgs := NewDriverGroupStorage(db)

	id := addGroup(t, dgs, DriverGroup{Name: "Original", Type: Network, MutuallyExclusive: false})

	group, _ := dgs.Get(id)
	group.Name = "Updated"
	group.Type = Display
	group.MutuallyExclusive = true

	if err := dgs.Update(group); err != nil {
		t.Fatalf("Update: %v", err)
	}

	updated, err := dgs.Get(id)
	if err != nil {
		t.Fatalf("Get after update: %v", err)
	}
	if updated.Name != "Updated" || updated.Type != Display || !updated.MutuallyExclusive {
		t.Errorf("unexpected updated values: %+v", updated)
	}
}

func TestDriverGroupStorage_Update_RemoveDriverCleansIncompatibles(t *testing.T) {
	db := openTestDB(t)
	dgs := NewDriverGroupStorage(db)

	groupId := addGroup(t, dgs, DriverGroup{
		Name:    "Incompat Group",
		Type:    Network,
		Drivers: []*Driver{{Name: "D1"}, {Name: "D2"}},
	})

	group, _ := dgs.Get(groupId)
	d1Id := group.Drivers[0].Id
	d2Id := group.Drivers[1].Id

	group.Drivers[0].IncompatibleIds = []uint{d2Id}
	group.Drivers[1].IncompatibleIds = []uint{d1Id}
	if err := dgs.Update(group); err != nil {
		t.Fatalf("Update with incompatibles: %v", err)
	}

	// Simulate UI removing D2: clear the reference from D1 before updating
	var d1 *Driver
	for _, d := range group.Drivers {
		if d.Id == d1Id {
			d.IncompatibleIds = nil
			d1 = d
			break
		}
	}
	group.Drivers = []*Driver{d1}
	if err := dgs.Update(group); err != nil {
		t.Fatalf("Update removing D2: %v", err)
	}

	updated, err := dgs.Get(groupId)
	if err != nil {
		t.Fatalf("Get after D2 removal: %v", err)
	}
	if len(updated.Drivers) != 1 {
		t.Fatalf("expected 1 driver, got %d", len(updated.Drivers))
	}
	if len(updated.Drivers[0].IncompatibleIds) != 0 {
		t.Errorf("expected empty incompatibles after D2 removal, got %v",
			updated.Drivers[0].IncompatibleIds)
	}
}

func TestDriverGroupStorage_Update_AddNewDriver(t *testing.T) {
	db := openTestDB(t)
	dgs := NewDriverGroupStorage(db)

	groupId := addGroup(t, dgs, DriverGroup{
		Name:    "G",
		Type:    Network,
		Drivers: []*Driver{{Name: "D1"}},
	})

	group, _ := dgs.Get(groupId)
	group.Drivers = append(group.Drivers, &Driver{Name: "D2"})
	if err := dgs.Update(group); err != nil {
		t.Fatalf("Update with new driver: %v", err)
	}

	updated, err := dgs.Get(groupId)
	if err != nil {
		t.Fatalf("Get after update: %v", err)
	}
	if len(updated.Drivers) != 2 {
		t.Fatalf("expected 2 drivers, got %d", len(updated.Drivers))
	}
	for _, d := range updated.Drivers {
		if d.Id == 0 {
			t.Error("every driver should have a non-zero ID after save")
		}
	}
}

func TestDriverGroupStorage_Remove_DeletesGroup(t *testing.T) {
	db := openTestDB(t)
	dgs := NewDriverGroupStorage(db)

	id1 := addGroup(t, dgs, DriverGroup{Name: "G1", Type: Network})
	id2 := addGroup(t, dgs, DriverGroup{Name: "G2", Type: Display})

	if err := dgs.Remove(id1); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	all, _ := dgs.All()
	if len(all) != 1 || all[0].Id != id2 {
		t.Errorf("expected only G2 remaining, got %v", all)
	}
}

func TestDriverGroupStorage_Remove_NotFound(t *testing.T) {
	db := openTestDB(t)
	dgs := NewDriverGroupStorage(db)

	if err := dgs.Remove(9999); err == nil {
		t.Fatal("expected error removing nonexistent group, got nil")
	}
}

func TestDriverGroupStorage_Remove_CascadeIncompatibles(t *testing.T) {
	db := openTestDB(t)
	dgs := NewDriverGroupStorage(db)

	g1Id := addGroup(t, dgs, DriverGroup{
		Name:    "G1",
		Type:    Network,
		Drivers: []*Driver{{Name: "D1"}},
	})
	g2Id := addGroup(t, dgs, DriverGroup{
		Name:    "G2",
		Type:    Display,
		Drivers: []*Driver{{Name: "D2"}},
	})

	g1, _ := dgs.Get(g1Id)
	g2, _ := dgs.Get(g2Id)
	d1Id := g1.Drivers[0].Id

	// D2 marks D1 as incompatible
	g2.Drivers[0].IncompatibleIds = []uint{d1Id}
	dgs.Update(g2)

	// Remove G1 — D1 is cascade-deleted; M2M rows referencing D1 are cascade-removed
	if err := dgs.Remove(g1Id); err != nil {
		t.Fatalf("Remove G1: %v", err)
	}

	g2After, err := dgs.Get(g2Id)
	if err != nil {
		t.Fatalf("Get G2 after removal: %v", err)
	}
	for _, d := range g2After.Drivers {
		for _, incId := range d.IncompatibleIds {
			if incId == d1Id {
				t.Errorf("d1Id %d still in IncompatibleIds after G1 removed", d1Id)
			}
		}
	}
}

func TestDriverGroupStorage_Add_PersistsGlobalExclusive(t *testing.T) {
	db := openTestDB(t)
	dgs := NewDriverGroupStorage(db)

	id := addGroup(t, dgs, DriverGroup{
		Name:    "Exclusive Group",
		Type:    Network,
		Drivers: []*Driver{{Name: "Solo", GlobalExclusive: true}},
	})

	group, err := dgs.Get(id)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(group.Drivers) != 1 {
		t.Fatalf("expected 1 driver, got %d", len(group.Drivers))
	}
	if !group.Drivers[0].GlobalExclusive {
		t.Errorf("expected GlobalExclusive=true after Add/Get round-trip")
	}
}

func TestDriverGroupStorage_Update_FlipsGlobalExclusive(t *testing.T) {
	db := openTestDB(t)
	dgs := NewDriverGroupStorage(db)

	id := addGroup(t, dgs, DriverGroup{
		Name:    "G",
		Type:    Network,
		Drivers: []*Driver{{Name: "D1", GlobalExclusive: false}},
	})

	group, err := dgs.Get(id)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	group.Drivers[0].GlobalExclusive = true
	if err := dgs.Update(group); err != nil {
		t.Fatalf("Update: %v", err)
	}

	updated, err := dgs.Get(id)
	if err != nil {
		t.Fatalf("Get after update: %v", err)
	}
	if !updated.Drivers[0].GlobalExclusive {
		t.Errorf("expected GlobalExclusive=true after Update")
	}

	// Reverse: true → false, locking in the symmetric bool-zero path.
	updated.Drivers[0].GlobalExclusive = false
	if err := dgs.Update(updated); err != nil {
		t.Fatalf("Update true→false: %v", err)
	}

	reverted, err := dgs.Get(id)
	if err != nil {
		t.Fatalf("Get after true→false update: %v", err)
	}
	if reverted.Drivers[0].GlobalExclusive {
		t.Errorf("expected GlobalExclusive=false after true→false Update")
	}
}

func TestDriverGroupStorage_Clone_PreservesGlobalExclusive(t *testing.T) {
	db := openTestDB(t)
	dgs := NewDriverGroupStorage(db)

	id := addGroup(t, dgs, DriverGroup{
		Name:    "Original",
		Type:    Network,
		Drivers: []*Driver{{Name: "D1", Path: "C:\\d1.exe", GlobalExclusive: true}},
	})

	if err := dgs.Clone(id); err != nil {
		t.Fatalf("Clone: %v", err)
	}

	all, err := dgs.All()
	if err != nil {
		t.Fatalf("All: %v", err)
	}
	var clone *DriverGroup
	for i := range all {
		if all[i].Name == "Original (copy)" {
			clone = &all[i]
			break
		}
	}
	if clone == nil {
		t.Fatal("clone group not found")
	}
	if len(clone.Drivers) != 1 {
		t.Fatalf("expected 1 cloned driver, got %d", len(clone.Drivers))
	}
	d := clone.Drivers[0]
	if !d.GlobalExclusive {
		t.Errorf("expected cloned GlobalExclusive=true")
	}
	if d.Name != "D1" || d.Path != "C:\\d1.exe" {
		t.Errorf("clone dropped normal fields: %+v", d)
	}
}

func TestDriverGroupStorage_MoveBehind_Forward(t *testing.T) {
	db := openTestDB(t)
	dgs := NewDriverGroupStorage(db)

	ids := make([]uint, 4)
	for i, name := range []string{"A", "B", "C", "D"} {
		ids[i] = addGroup(t, dgs, DriverGroup{Name: name, Type: Network})
	}

	// Move A (position 0) to behind index 2 → [B, C, D, A]
	if err := dgs.MoveBehind(ids[0], 2); err != nil {
		t.Fatalf("MoveBehind: %v", err)
	}

	result, err := dgs.All()
	if err != nil {
		t.Fatalf("All after MoveBehind: %v", err)
	}
	if len(result) != 4 {
		t.Fatalf("expected 4 groups, got %d", len(result))
	}
	if result[3].Id != ids[0] {
		t.Errorf("expected A at index 3, got id=%d", result[3].Id)
	}
	if result[0].Id != ids[1] {
		t.Errorf("expected B at index 0, got id=%d", result[0].Id)
	}
}

func TestDriverGroupStorage_MoveBehind_Backward(t *testing.T) {
	db := openTestDB(t)
	dgs := NewDriverGroupStorage(db)

	ids := make([]uint, 4)
	for i, name := range []string{"A", "B", "C", "D"} {
		ids[i] = addGroup(t, dgs, DriverGroup{Name: name, Type: Network})
	}

	// Move D (position 3) to behind index 0 → [A, D, B, C]
	if err := dgs.MoveBehind(ids[3], 0); err != nil {
		t.Fatalf("MoveBehind: %v", err)
	}

	result, err := dgs.All()
	if err != nil {
		t.Fatalf("All after MoveBehind: %v", err)
	}
	if result[0].Id != ids[0] {
		t.Errorf("expected A at index 0, got id=%d", result[0].Id)
	}
	if result[1].Id != ids[3] {
		t.Errorf("expected D at index 1, got id=%d", result[1].Id)
	}
}

func TestDriverGroupStorage_MoveBehind_NegativeIndex(t *testing.T) {
	db := openTestDB(t)
	dgs := NewDriverGroupStorage(db)

	id := addGroup(t, dgs, DriverGroup{Name: "G1", Type: Network})
	addGroup(t, dgs, DriverGroup{Name: "G2", Type: Display})

	if err := dgs.MoveBehind(id, -1); err != nil {
		t.Fatalf("MoveBehind(-1): %v", err)
	}

	all, _ := dgs.All()
	if len(all) != 2 {
		t.Errorf("expected 2 groups unchanged, got %d", len(all))
	}
}

func TestDriverGroupStorage_CompleteWorkflow(t *testing.T) {
	db := openTestDB(t)
	dgs := NewDriverGroupStorage(db)

	id1 := addGroup(t, dgs, DriverGroup{Name: "Network", Type: Network,
		Drivers: []*Driver{{Name: "Net Driver"}}})
	id2 := addGroup(t, dgs, DriverGroup{Name: "Display", Type: Display,
		Drivers: []*Driver{{Name: "GPU Driver"}}})

	all, _ := dgs.All()
	if len(all) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(all))
	}

	g, _ := dgs.Get(id1)
	g.Name = "Updated Network"
	dgs.Update(g)

	g, _ = dgs.Get(id1)
	if g.Name != "Updated Network" {
		t.Errorf("expected updated name, got %s", g.Name)
	}

	dgs.MoveBehind(id1, 0)

	all, _ = dgs.All()
	if all[1].Id != id1 {
		t.Errorf("expected id1 at index 1 after MoveBehind, got %d", all[1].Id)
	}

	dgs.Remove(id2)
	all, _ = dgs.All()
	if len(all) != 1 {
		t.Errorf("expected 1 group after remove, got %d", len(all))
	}
}
