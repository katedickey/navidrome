package persistence

import (
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("PlaylistRepository", func() {
	var repo model.PlaylistRepository

	BeforeEach(func() {
		ctx := log.NewContext(GinkgoT().Context())
		ctx = request.WithUser(ctx, model.User{ID: "userid", UserName: "userid", IsAdmin: true})
		repo = NewPlaylistRepository(ctx, GetDBXBuilder())
	})

	Describe("Count", func() {
		It("returns the number of playlists in the DB", func() {
			Expect(repo.CountAll()).To(Equal(int64(2)))
		})
	})

	Describe("Exists", func() {
		It("returns true for an existing playlist", func() {
			Expect(repo.Exists(plsCool.ID)).To(BeTrue())
		})
		It("returns false for a non-existing playlist", func() {
			Expect(repo.Exists("666")).To(BeFalse())
		})
	})

	Describe("Get", func() {
		It("returns an existing playlist", func() {
			p, err := repo.Get(plsBest.ID)
			Expect(err).To(BeNil())
			// Compare all but Tracks and timestamps
			p2 := *p
			p2.Tracks = plsBest.Tracks
			p2.UpdatedAt = plsBest.UpdatedAt
			p2.CreatedAt = plsBest.CreatedAt
			Expect(p2).To(Equal(plsBest))
			// Compare tracks
			for i := range p.Tracks {
				Expect(p.Tracks[i].ID).To(Equal(plsBest.Tracks[i].ID))
			}
		})
		It("returns ErrNotFound for a non-existing playlist", func() {
			_, err := repo.Get("666")
			Expect(err).To(MatchError(model.ErrNotFound))
		})
		It("returns all tracks", func() {
			pls, err := repo.GetWithTracks(plsBest.ID, true, false)
			Expect(err).ToNot(HaveOccurred())
			Expect(pls.Name).To(Equal(plsBest.Name))
			Expect(pls.Tracks).To(HaveLen(2))
			Expect(pls.Tracks[0].ID).To(Equal("1"))
			Expect(pls.Tracks[0].PlaylistID).To(Equal(plsBest.ID))
			Expect(pls.Tracks[0].MediaFileID).To(Equal(songDayInALife.ID))
			Expect(pls.Tracks[0].MediaFile.ID).To(Equal(songDayInALife.ID))
			Expect(pls.Tracks[1].ID).To(Equal("2"))
			Expect(pls.Tracks[1].PlaylistID).To(Equal(plsBest.ID))
			Expect(pls.Tracks[1].MediaFileID).To(Equal(songRadioactivity.ID))
			Expect(pls.Tracks[1].MediaFile.ID).To(Equal(songRadioactivity.ID))
			mfs := pls.MediaFiles()
			Expect(mfs).To(HaveLen(2))
			Expect(mfs[0].ID).To(Equal(songDayInALife.ID))
			Expect(mfs[1].ID).To(Equal(songRadioactivity.ID))
		})
	})

	It("Put/Exists/Delete", func() {
		By("saves the playlist to the DB")
		newPls := model.Playlist{Name: "Great!", OwnerID: "userid"}
		newPls.AddMediaFilesByID([]string{"1004", "1003"})

		By("saves the playlist to the DB")
		Expect(repo.Put(&newPls)).To(BeNil())

		By("adds repeated songs to a playlist and keeps the order")
		newPls.AddMediaFilesByID([]string{"1004"})
		Expect(repo.Put(&newPls)).To(BeNil())
		saved, _ := repo.GetWithTracks(newPls.ID, true, false)
		Expect(saved.Tracks).To(HaveLen(3))
		Expect(saved.Tracks[0].MediaFileID).To(Equal("1004"))
		Expect(saved.Tracks[1].MediaFileID).To(Equal("1003"))
		Expect(saved.Tracks[2].MediaFileID).To(Equal("1004"))

		By("returns the newly created playlist")
		Expect(repo.Exists(newPls.ID)).To(BeTrue())

		By("returns deletes the playlist")
		Expect(repo.Delete(newPls.ID)).To(BeNil())

		By("returns error if tries to retrieve the deleted playlist")
		Expect(repo.Exists(newPls.ID)).To(BeFalse())
	})

	Describe("GetAll", func() {
		It("returns all playlists from DB", func() {
			all, err := repo.GetAll()
			Expect(err).To(BeNil())
			Expect(all[0].ID).To(Equal(plsBest.ID))
			Expect(all[1].ID).To(Equal(plsCool.ID))
		})
	})

	Describe("GetPlaylists", func() {
		It("returns playlists for a track", func() {
			pls, err := repo.GetPlaylists(songRadioactivity.ID)
			Expect(err).ToNot(HaveOccurred())
			Expect(pls).To(HaveLen(1))
			Expect(pls[0].ID).To(Equal(plsBest.ID))
		})

		It("returns empty when none", func() {
			pls, err := repo.GetPlaylists("9999")
			Expect(err).ToNot(HaveOccurred())
			Expect(pls).To(HaveLen(0))
		})
	})

	Describe("Track Deletion and Renumbering", func() {
		var testPlaylistID string

		AfterEach(func() {
			if testPlaylistID != "" {
				Expect(repo.Delete(testPlaylistID)).To(BeNil())
				testPlaylistID = ""
			}
		})

		// helper to get track positions and media file IDs
		getTrackInfo := func(playlistID string) (ids []string, mediaFileIDs []string) {
			pls, err := repo.GetWithTracks(playlistID, false, false)
			Expect(err).ToNot(HaveOccurred())
			for _, t := range pls.Tracks {
				ids = append(ids, t.ID)
				mediaFileIDs = append(mediaFileIDs, t.MediaFileID)
			}
			return
		}

		It("renumbers correctly after deleting a track from the middle", func() {
			By("creating a playlist with 4 tracks")
			newPls := model.Playlist{Name: "Renumber Test Middle", OwnerID: "userid"}
			newPls.AddMediaFilesByID([]string{"1001", "1002", "1003", "1004"})
			Expect(repo.Put(&newPls)).To(Succeed())
			testPlaylistID = newPls.ID

			By("deleting the second track (position 2)")
			tracksRepo := repo.Tracks(newPls.ID, false)
			Expect(tracksRepo.Delete("2")).To(Succeed())

			By("verifying remaining tracks are renumbered sequentially")
			ids, mediaFileIDs := getTrackInfo(newPls.ID)
			Expect(ids).To(Equal([]string{"1", "2", "3"}))
			Expect(mediaFileIDs).To(Equal([]string{"1001", "1003", "1004"}))
		})

		It("renumbers correctly after deleting the first track", func() {
			By("creating a playlist with 3 tracks")
			newPls := model.Playlist{Name: "Renumber Test First", OwnerID: "userid"}
			newPls.AddMediaFilesByID([]string{"1001", "1002", "1003"})
			Expect(repo.Put(&newPls)).To(Succeed())
			testPlaylistID = newPls.ID

			By("deleting the first track (position 1)")
			tracksRepo := repo.Tracks(newPls.ID, false)
			Expect(tracksRepo.Delete("1")).To(Succeed())

			By("verifying remaining tracks are renumbered sequentially")
			ids, mediaFileIDs := getTrackInfo(newPls.ID)
			Expect(ids).To(Equal([]string{"1", "2"}))
			Expect(mediaFileIDs).To(Equal([]string{"1002", "1003"}))
		})

		It("renumbers correctly after deleting the last track", func() {
			By("creating a playlist with 3 tracks")
			newPls := model.Playlist{Name: "Renumber Test Last", OwnerID: "userid"}
			newPls.AddMediaFilesByID([]string{"1001", "1002", "1003"})
			Expect(repo.Put(&newPls)).To(Succeed())
			testPlaylistID = newPls.ID

			By("deleting the last track (position 3)")
			tracksRepo := repo.Tracks(newPls.ID, false)
			Expect(tracksRepo.Delete("3")).To(Succeed())

			By("verifying remaining tracks are renumbered sequentially")
			ids, mediaFileIDs := getTrackInfo(newPls.ID)
			Expect(ids).To(Equal([]string{"1", "2"}))
			Expect(mediaFileIDs).To(Equal([]string{"1001", "1002"}))
		})
	})

	Describe("Visibility", func() {
		var privatePls model.Playlist
		var regularRepo, thirdRepo model.PlaylistRepository

		playlistIDs := func(list model.Playlists) []string {
			ids := make([]string, len(list))
			for i, p := range list {
				ids[i] = p.ID
			}
			return ids
		}

		BeforeEach(func() {
			regularCtx := log.NewContext(GinkgoT().Context())
			regularCtx = request.WithUser(regularCtx, regularUser)
			regularRepo = NewPlaylistRepository(regularCtx, GetDBXBuilder())

			thirdCtx := log.NewContext(GinkgoT().Context())
			thirdCtx = request.WithUser(thirdCtx, thirdUser)
			thirdRepo = NewPlaylistRepository(thirdCtx, GetDBXBuilder())

			privatePls = model.Playlist{Name: "Visibility Test", OwnerID: regularUser.ID, OwnerName: regularUser.UserName}
			privatePls.AddMediaFilesByID([]string{songAntenna2.ID})
			Expect(regularRepo.Put(&privatePls)).To(Succeed())
		})

		AfterEach(func() {
			Expect(repo.Delete(privatePls.ID)).To(Succeed())
		})

		It("hides the playlist from a user with no grant", func() {
			_, err := thirdRepo.Get(privatePls.ID)
			Expect(err).To(MatchError(model.ErrNotFound))
			Expect(thirdRepo.CountAll()).To(Equal(int64(1))) // only the public plsBest
			Expect(thirdRepo.Exists(privatePls.ID)).To(BeFalse())

			pls, err := thirdRepo.GetPlaylists(songAntenna2.ID)
			Expect(err).ToNot(HaveOccurred())
			Expect(pls).To(BeEmpty())

			all, err := thirdRepo.GetAll()
			Expect(err).ToNot(HaveOccurred())
			Expect(playlistIDs(all)).ToNot(ContainElement(privatePls.ID))
		})

		It("exposes the playlist to a user granted visibility", func() {
			Expect(repo.SetVisibleUsers(privatePls.ID, []string{thirdUser.ID})).To(Succeed())

			p, err := thirdRepo.Get(privatePls.ID)
			Expect(err).ToNot(HaveOccurred())
			Expect(p.ID).To(Equal(privatePls.ID))

			Expect(thirdRepo.CountAll()).To(Equal(int64(2)))
			Expect(thirdRepo.Exists(privatePls.ID)).To(BeTrue())

			pls, err := thirdRepo.GetPlaylists(songAntenna2.ID)
			Expect(err).ToNot(HaveOccurred())
			Expect(pls).To(HaveLen(1))
			Expect(pls[0].ID).To(Equal(privatePls.ID))

			all, err := thirdRepo.GetAll()
			Expect(err).ToNot(HaveOccurred())
			Expect(playlistIDs(all)).To(ContainElement(privatePls.ID))
		})

		It("keeps admin visibility unaffected by visibility grants (regression)", func() {
			p, err := repo.Get(privatePls.ID)
			Expect(err).ToNot(HaveOccurred())
			Expect(p.ID).To(Equal(privatePls.ID))
		})

		It("keeps owner visibility unaffected by visibility grants (regression)", func() {
			p, err := regularRepo.Get(privatePls.ID)
			Expect(err).ToNot(HaveOccurred())
			Expect(p.ID).To(Equal(privatePls.ID))
		})

		It("replaces the visibility set instead of appending to it", func() {
			Expect(repo.SetVisibleUsers(privatePls.ID, []string{thirdUser.ID})).To(Succeed())
			Expect(repo.SetVisibleUsers(privatePls.ID, []string{adminUser.ID})).To(Succeed())

			users, err := repo.GetVisibleUsers(privatePls.ID)
			Expect(err).ToNot(HaveOccurred())
			Expect(users).To(HaveLen(1))
			Expect(users[0].ID).To(Equal(adminUser.ID))
		})

		It("clears the visibility set when given an empty slice", func() {
			Expect(repo.SetVisibleUsers(privatePls.ID, []string{thirdUser.ID, adminUser.ID})).To(Succeed())
			Expect(repo.SetVisibleUsers(privatePls.ID, nil)).To(Succeed())

			users, err := repo.GetVisibleUsers(privatePls.ID)
			Expect(err).ToNot(HaveOccurred())
			Expect(users).To(BeEmpty())
		})

		It("round-trips the visibility set via GetVisibleUsers", func() {
			Expect(repo.SetVisibleUsers(privatePls.ID, []string{thirdUser.ID, adminUser.ID})).To(Succeed())

			users, err := repo.GetVisibleUsers(privatePls.ID)
			Expect(err).ToNot(HaveOccurred())
			Expect(users).To(HaveLen(2))
			ids := []string{users[0].ID, users[1].ID}
			Expect(ids).To(ConsistOf(thirdUser.ID, adminUser.ID))
		})

		It("does not let a user with only visibility delete the playlist", func() {
			Expect(repo.SetVisibleUsers(privatePls.ID, []string{thirdUser.ID})).To(Succeed())
			Expect(thirdRepo.Delete(privatePls.ID)).To(Succeed())
			Expect(repo.Exists(privatePls.ID)).To(BeTrue())
		})

		It("lets the owner delete the playlist", func() {
			Expect(regularRepo.Delete(privatePls.ID)).To(Succeed())
			Expect(repo.Exists(privatePls.ID)).To(BeFalse())
		})

		It("lets an admin delete the playlist", func() {
			Expect(repo.Delete(privatePls.ID)).To(Succeed())
			Expect(repo.Exists(privatePls.ID)).To(BeFalse())
		})
	})
})
