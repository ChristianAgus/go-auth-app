package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"auth-app/internal/model"
	"auth-app/internal/repository"
)

type fakeNoteStore struct {
	notes  map[uint64]*model.Note
	nextID uint64

	// dicatat supaya test bisa ngecek nilai yang diterima repository
	lastLimit  int
	lastOffset int
}

func newFakeNoteStore() *fakeNoteStore {
	return &fakeNoteStore{notes: map[uint64]*model.Note{}, nextID: 1}
}

func (f *fakeNoteStore) Create(_ context.Context, n *model.Note) error {
	n.ID = f.nextID
	f.nextID++
	now := time.Now()
	n.CreatedAt, n.UpdatedAt = now, now

	stored := *n
	f.notes[n.ID] = &stored
	return nil
}

func (f *fakeNoteStore) GetByID(_ context.Context, id, userID uint64) (*model.Note, error) {
	n, ok := f.notes[id]
	if !ok || n.UserID != userID {
		return nil, repository.ErrNotFound
	}
	c := *n
	return &c, nil
}

func (f *fakeNoteStore) List(_ context.Context, userID uint64, search string, limit, offset int) ([]model.Note, int, error) {
	f.lastLimit, f.lastOffset = limit, offset

	result := make([]model.Note, 0)
	for id := uint64(1); id < f.nextID; id++ {
		n, ok := f.notes[id]
		if !ok || n.UserID != userID {
			continue
		}
		if search != "" && !strings.Contains(n.Title, search) && !strings.Contains(n.Content, search) {
			continue
		}
		result = append(result, *n)
	}

	total := len(result)
	if offset > total {
		offset = total
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return result[offset:end], total, nil
}

func (f *fakeNoteStore) Update(_ context.Context, n *model.Note) error {
	stored, ok := f.notes[n.ID]
	if !ok || stored.UserID != n.UserID {
		return nil
	}
	stored.Title = n.Title
	stored.Content = n.Content
	return nil
}

func (f *fakeNoteStore) Delete(_ context.Context, id, userID uint64) error {
	n, ok := f.notes[id]
	if !ok || n.UserID != userID {
		return repository.ErrNotFound
	}
	delete(f.notes, id)
	return nil
}

func newTestNoteService() (*NoteService, *fakeNoteStore) {
	store := newFakeNoteStore()
	return NewNoteService(store), store
}

func TestNoteCreate_Validation(t *testing.T) {
	svc, _ := newTestNoteService()

	tests := []struct {
		name    string
		title   string
		content string
	}{
		{"judul kosong", "", "isi"},
		{"judul cuma spasi", "   ", "isi"},
		{"judul 201 karakter", strings.Repeat("a", 201), "isi"},
		{"isi lebih dari 10000 karakter", "judul", strings.Repeat("a", 10001)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.Create(context.Background(), 1, tt.title, tt.content)

			var ve *ValidationError
			if !errors.As(err, &ve) {
				t.Fatalf("harusnya ValidationError, dapat: %v", err)
			}
		})
	}
}

func TestNoteCreate_CountsCharactersNotBytes(t *testing.T) {
	svc, _ := newTestNoteService()
	ctx := context.Background()

	// "é" = 2 byte, tapi 1 karakter. 200 karakter harus lolos.
	if _, err := svc.Create(ctx, 1, strings.Repeat("é", 200), "isi"); err != nil {
		t.Errorf("200 karakter harusnya lolos: %v", err)
	}
	if _, err := svc.Create(ctx, 1, strings.Repeat("é", 201), "isi"); err == nil {
		t.Error("201 karakter harusnya ditolak")
	}
}

func TestNoteCreate_TrimsTitle(t *testing.T) {
	svc, _ := newTestNoteService()

	note, err := svc.Create(context.Background(), 1, "  Belajar Go  ", "isi")
	if err != nil {
		t.Fatal(err)
	}
	if note.Title != "Belajar Go" {
		t.Errorf("judul harusnya di-trim, dapat %q", note.Title)
	}
	if note.ID == 0 {
		t.Error("ID harusnya terisi")
	}
}

// Ini test terpenting: user lain TIDAK boleh menyentuh catatan orang.
func TestNote_OwnershipIsolation(t *testing.T) {
	svc, _ := newTestNoteService()
	ctx := context.Background()

	const owner, intruder = uint64(1), uint64(2)

	note, err := svc.Create(ctx, owner, "Rahasia Chris", "isi rahasia")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := svc.Get(ctx, intruder, note.ID); !errors.Is(err, ErrNoteNotFound) {
		t.Errorf("Get oleh orang lain harusnya ErrNoteNotFound, dapat: %v", err)
	}
	if _, err := svc.Update(ctx, intruder, note.ID, "dibajak", "dibajak"); !errors.Is(err, ErrNoteNotFound) {
		t.Errorf("Update oleh orang lain harusnya ErrNoteNotFound, dapat: %v", err)
	}
	if err := svc.Delete(ctx, intruder, note.ID); !errors.Is(err, ErrNoteNotFound) {
		t.Errorf("Delete oleh orang lain harusnya ErrNoteNotFound, dapat: %v", err)
	}

	// catatan asli harus masih utuh
	got, err := svc.Get(ctx, owner, note.ID)
	if err != nil {
		t.Fatalf("catatan pemilik hilang: %v", err)
	}
	if got.Title != "Rahasia Chris" {
		t.Errorf("catatan berubah: %q", got.Title)
	}
}

func TestNoteUpdate(t *testing.T) {
	svc, _ := newTestNoteService()
	ctx := context.Background()

	note, _ := svc.Create(ctx, 1, "Lama", "isi lama")

	updated, err := svc.Update(ctx, 1, note.ID, "Baru", "isi baru")
	if err != nil {
		t.Fatal(err)
	}
	if updated.Title != "Baru" || updated.Content != "isi baru" {
		t.Errorf("update tidak tersimpan: %+v", updated)
	}

	// validasi tetap jalan saat update
	_, err = svc.Update(ctx, 1, note.ID, "", "isi")
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Errorf("judul kosong harusnya ValidationError, dapat: %v", err)
	}
}

func TestNoteDelete(t *testing.T) {
	svc, _ := newTestNoteService()
	ctx := context.Background()

	note, _ := svc.Create(ctx, 1, "Hapus aku", "isi")

	if err := svc.Delete(ctx, 1, note.ID); err != nil {
		t.Fatalf("delete gagal: %v", err)
	}
	if _, err := svc.Get(ctx, 1, note.ID); !errors.Is(err, ErrNoteNotFound) {
		t.Errorf("setelah dihapus harusnya ErrNoteNotFound, dapat: %v", err)
	}
	if err := svc.Delete(ctx, 1, note.ID); !errors.Is(err, ErrNoteNotFound) {
		t.Errorf("hapus dua kali harusnya ErrNoteNotFound, dapat: %v", err)
	}
}

func TestNoteList_OnlyOwnNotesAndSearch(t *testing.T) {
	svc, _ := newTestNoteService()
	ctx := context.Background()

	svc.Create(ctx, 1, "Belajar Go", "migration")
	svc.Create(ctx, 1, "Resep nasi goreng", "bawang")
	svc.Create(ctx, 2, "Catatan Budi", "punya budi")

	notes, total, err := svc.List(ctx, 1, "", 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 || len(notes) != 2 {
		t.Errorf("user 1 harusnya punya 2 catatan, dapat total=%d len=%d", total, len(notes))
	}

	_, total, _ = svc.List(ctx, 1, "Belajar", 1, 10)
	if total != 1 {
		t.Errorf("search 'Belajar' harusnya 1 hasil, dapat %d", total)
	}

	_, total, _ = svc.List(ctx, 2, "", 1, 10)
	if total != 1 {
		t.Errorf("user 2 harusnya cuma lihat 1 catatan miliknya, dapat %d", total)
	}
}

func TestNoteList_PaginationLimits(t *testing.T) {
	tests := []struct {
		name       string
		page       int
		limit      int
		wantLimit  int
		wantOffset int
	}{
		{"default saat 0", 0, 0, 10, 0},
		{"page negatif jadi 1", -5, 10, 10, 0},
		{"limit dibatasi 100", 1, 1000, 100, 0},
		{"halaman 3, limit 5", 3, 5, 5, 10},
		{"halaman 2, limit default", 2, 0, 10, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, store := newTestNoteService()

			if _, _, err := svc.List(context.Background(), 1, "", tt.page, tt.limit); err != nil {
				t.Fatal(err)
			}
			if store.lastLimit != tt.wantLimit || store.lastOffset != tt.wantOffset {
				t.Errorf("limit/offset = %d/%d, harusnya %d/%d",
					store.lastLimit, store.lastOffset, tt.wantLimit, tt.wantOffset)
			}
		})
	}
}