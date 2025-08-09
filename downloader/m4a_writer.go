package downloader

import (
	"encoding/binary"
	"io"
	"strconv"

	"github.com/abema/go-mp4"
)

// WriteM4a writes the decrypted song data to an M4A file
func (sd *SongDownloaderImpl) WriteM4a(w *mp4.Writer, info *SongInfo, meta *AutoSong, data []byte) error {
	albums := meta.Relationships.Albums.Data
	artists := meta.Relationships.Artists.Data
	{ // ftyp
		box, err := w.StartBox(&mp4.BoxInfo{Type: mp4.BoxTypeFtyp()})
		if err != nil {
			return err
		}
		_, err = mp4.Marshal(w, &mp4.Ftyp{
			MajorBrand:   [4]byte{'M', '4', 'A', ' '},
			MinorVersion: 0,
			CompatibleBrands: []mp4.CompatibleBrandElem{
				{CompatibleBrand: [4]byte{'M', '4', 'A', ' '}},
				{CompatibleBrand: [4]byte{'m', 'p', '4', '2'}},
				{CompatibleBrand: mp4.BrandISOM()},
				{CompatibleBrand: [4]byte{0, 0, 0, 0}},
			},
		}, box.Context)
		if err != nil {
			return err
		}
		_, err = w.EndBox()
		if err != nil {
			return err
		}
	}

	const chunkSize uint32 = 5
	duration := info.Duration()
	numSamples := uint32(len(info.samples))
	var stco *mp4.BoxInfo

	{ // moov
		_, err := w.StartBox(&mp4.BoxInfo{Type: mp4.BoxTypeMoov()})
		if err != nil {
			return err
		}
		box, err := mp4.ExtractBox(info.r, nil, mp4.BoxPath{mp4.BoxTypeMoov()})
		if err != nil {
			return err
		}
		moovOri := box[0]

		{ // mvhd
			_, err = w.StartBox(&mp4.BoxInfo{Type: mp4.BoxTypeMvhd()})
			if err != nil {
				return err
			}

			oriBox, err := mp4.ExtractBoxWithPayload(info.r, moovOri, mp4.BoxPath{mp4.BoxTypeMvhd()})
			if err != nil {
				return err
			}
			mvhd := oriBox[0].Payload.(*mp4.Mvhd)
			if mvhd.Version == 0 {
				mvhd.DurationV0 = uint32(duration)
			} else {
				mvhd.DurationV1 = duration
			}

			_, err = mp4.Marshal(w, mvhd, oriBox[0].Info.Context)
			if err != nil {
				return err
			}

			_, err = w.EndBox()
			if err != nil {
				return err
			}
		}

		{ // trak
			_, err = w.StartBox(&mp4.BoxInfo{Type: mp4.BoxTypeTrak()})
			if err != nil {
				return err
			}

			box, err := mp4.ExtractBox(info.r, moovOri, mp4.BoxPath{mp4.BoxTypeTrak()})
			if err != nil {
				return err
			}
			trakOri := box[0]

			{ // tkhd
				_, err = w.StartBox(&mp4.BoxInfo{Type: mp4.BoxTypeTkhd()})
				if err != nil {
					return err
				}

				oriBox, err := mp4.ExtractBoxWithPayload(info.r, trakOri, mp4.BoxPath{mp4.BoxTypeTkhd()})
				if err != nil {
					return err
				}
				tkhd := oriBox[0].Payload.(*mp4.Tkhd)
				if tkhd.Version == 0 {
					tkhd.DurationV0 = uint32(duration)
				} else {
					tkhd.DurationV1 = duration
				}
				tkhd.SetFlags(0x7)

				_, err = mp4.Marshal(w, tkhd, oriBox[0].Info.Context)
				if err != nil {
					return err
				}

				_, err = w.EndBox()
				if err != nil {
					return err
				}
			}

			{ // mdia
				_, err = w.StartBox(&mp4.BoxInfo{Type: mp4.BoxTypeMdia()})
				if err != nil {
					return err
				}

				box, err := mp4.ExtractBox(info.r, trakOri, mp4.BoxPath{mp4.BoxTypeMdia()})
				if err != nil {
					return err
				}
				mdiaOri := box[0]

				{ // mdhd
					_, err = w.StartBox(&mp4.BoxInfo{Type: mp4.BoxTypeMdhd()})
					if err != nil {
						return err
					}

					oriBox, err := mp4.ExtractBoxWithPayload(info.r, mdiaOri, mp4.BoxPath{mp4.BoxTypeMdhd()})
					if err != nil {
						return err
					}
					mdhd := oriBox[0].Payload.(*mp4.Mdhd)
					if mdhd.Version == 0 {
						mdhd.DurationV0 = uint32(duration)
					} else {
						mdhd.DurationV1 = duration
					}

					_, err = mp4.Marshal(w, mdhd, oriBox[0].Info.Context)
					if err != nil {
						return err
					}

					_, err = w.EndBox()
					if err != nil {
						return err
					}
				}

				{ // hdlr
					oriBox, err := mp4.ExtractBox(info.r, mdiaOri, mp4.BoxPath{mp4.BoxTypeHdlr()})
					if err != nil {
						return err
					}

					err = w.CopyBox(info.r, oriBox[0])
					if err != nil {
						return err
					}
				}

				{ // minf
					_, err = w.StartBox(&mp4.BoxInfo{Type: mp4.BoxTypeMinf()})
					if err != nil {
						return err
					}

					box, err := mp4.ExtractBox(info.r, mdiaOri, mp4.BoxPath{mp4.BoxTypeMinf()})
					if err != nil {
						return err
					}
					minfOri := box[0]

					{ // smhd, dinf
						boxes, err := mp4.ExtractBoxes(info.r, minfOri, []mp4.BoxPath{
							{mp4.BoxTypeSmhd()},
							{mp4.BoxTypeDinf()},
						})
						if err != nil {
							return err
						}

						for _, b := range boxes {
							err = w.CopyBox(info.r, b)
							if err != nil {
								return err
							}
						}
					}

					{ // stbl
						_, err = w.StartBox(&mp4.BoxInfo{Type: mp4.BoxTypeStbl()})
						if err != nil {
							return err
						}

						{ // stsd
							box, err := w.StartBox(&mp4.BoxInfo{Type: mp4.BoxTypeStsd()})
							if err != nil {
								return err
							}
							_, err = mp4.Marshal(w, &mp4.Stsd{EntryCount: 1}, box.Context)
							if err != nil {
								return err
							}

							{ // alac
								_, err = w.StartBox(&mp4.BoxInfo{Type: BoxTypeAlac()})
								if err != nil {
									return err
								}

								_, err = w.Write([]byte{
									0, 0, 0, 0, 0, 0, 0, 1,
									0, 0, 0, 0, 0, 0, 0, 0})
								if err != nil {
									return err
								}

								err = binary.Write(w, binary.BigEndian, uint16(info.alacParam.NumChannels))
								if err != nil {
									return err
								}

								err = binary.Write(w, binary.BigEndian, uint16(info.alacParam.BitDepth))
								if err != nil {
									return err
								}

								_, err = w.Write([]byte{0, 0})
								if err != nil {
									return err
								}

								err = binary.Write(w, binary.BigEndian, info.alacParam.SampleRate)
								if err != nil {
									return err
								}

								_, err = w.Write([]byte{0, 0})
								if err != nil {
									return err
								}

								box, err := w.StartBox(&mp4.BoxInfo{Type: BoxTypeAlac()})
								if err != nil {
									return err
								}

								_, err = mp4.Marshal(w, info.alacParam, box.Context)
								if err != nil {
									return err
								}

								_, err = w.EndBox()
								if err != nil {
									return err
								}

								_, err = w.EndBox()
								if err != nil {
									return err
								}
							}

							_, err = w.EndBox()
							if err != nil {
								return err
							}
						}

						{ // stts
							box, err := w.StartBox(&mp4.BoxInfo{Type: mp4.BoxTypeStts()})
							if err != nil {
								return err
							}

							var stts mp4.Stts
							for _, sample := range info.samples {
								if len(stts.Entries) != 0 {
									last := &stts.Entries[len(stts.Entries)-1]
									if last.SampleDelta == sample.duration {
										last.SampleCount++
										continue
									}
								}
								stts.Entries = append(stts.Entries, mp4.SttsEntry{
									SampleCount: 1,
									SampleDelta: sample.duration,
								})
							}
							stts.EntryCount = uint32(len(stts.Entries))

							_, err = mp4.Marshal(w, &stts, box.Context)
							if err != nil {
								return err
							}

							_, err = w.EndBox()
							if err != nil {
								return err
							}
						}

						{ // stsc
							box, err := w.StartBox(&mp4.BoxInfo{Type: mp4.BoxTypeStsc()})
							if err != nil {
								return err
							}

							if numSamples%chunkSize == 0 {
								_, err = mp4.Marshal(w, &mp4.Stsc{
									EntryCount: 1,
									Entries: []mp4.StscEntry{
										{
											FirstChunk:             1,
											SamplesPerChunk:        chunkSize,
											SampleDescriptionIndex: 1,
										},
									},
								}, box.Context)
							} else {
								_, err = mp4.Marshal(w, &mp4.Stsc{
									EntryCount: 2,
									Entries: []mp4.StscEntry{
										{
											FirstChunk:             1,
											SamplesPerChunk:        chunkSize,
											SampleDescriptionIndex: 1,
										}, {
											FirstChunk:             numSamples/chunkSize + 1,
											SamplesPerChunk:        numSamples % chunkSize,
											SampleDescriptionIndex: 1,
										},
									},
								}, box.Context)
							}

							_, err = w.EndBox()
							if err != nil {
								return err
							}
						}

						{ // stsz
							box, err := w.StartBox(&mp4.BoxInfo{Type: mp4.BoxTypeStsz()})
							if err != nil {
								return err
							}

							stsz := mp4.Stsz{SampleCount: numSamples}
							for _, sample := range info.samples {
								stsz.EntrySize = append(stsz.EntrySize, uint32(len(sample.data)))
							}

							_, err = mp4.Marshal(w, &stsz, box.Context)
							if err != nil {
								return err
							}

							_, err = w.EndBox()
							if err != nil {
								return err
							}
						}

						{ // stco
							box, err := w.StartBox(&mp4.BoxInfo{Type: mp4.BoxTypeStco()})
							if err != nil {
								return err
							}

							l := (numSamples + chunkSize - 1) / chunkSize
							_, err = mp4.Marshal(w, &mp4.Stco{
								EntryCount:  l,
								ChunkOffset: make([]uint32, l),
							}, box.Context)

							stco, err = w.EndBox()
							if err != nil {
								return err
							}
						}

						_, err = w.EndBox()
						if err != nil {
							return err
						}
					}

					_, err = w.EndBox()
					if err != nil {
						return err
					}
				}

				_, err = w.EndBox()
				if err != nil {
					return err
				}
			}

			_, err = w.EndBox()
			if err != nil {
				return err
			}
		}

		{ // udta
			ctx := mp4.Context{UnderUdta: true}
			_, err := w.StartBox(&mp4.BoxInfo{Type: mp4.BoxTypeUdta(), Context: ctx})
			if err != nil {
				return err
			}

			{ // meta
				ctx.UnderIlstMeta = true

				_, err = w.StartBox(&mp4.BoxInfo{Type: mp4.BoxTypeMeta(), Context: ctx})
				if err != nil {
					return err
				}

				_, err = mp4.Marshal(w, &mp4.Meta{}, ctx)
				if err != nil {
					return err
				}

				{ // hdlr
					_, err = w.StartBox(&mp4.BoxInfo{Type: mp4.BoxTypeHdlr(), Context: ctx})
					if err != nil {
						return err
					}

					_, err = mp4.Marshal(w, &mp4.Hdlr{
						HandlerType: [4]byte{'m', 'd', 'i', 'r'},
						Reserved:    [3]uint32{0x6170706c, 0, 0},
					}, ctx)
					if err != nil {
						return err
					}

					_, err = w.EndBox()
					if err != nil {
						return err
					}
				}

				{ // ilst
					ctx.UnderIlst = true

					_, err = w.StartBox(&mp4.BoxInfo{Type: mp4.BoxTypeIlst(), Context: ctx})
					if err != nil {
						return err
					}

					marshalData := func(val interface{}) error {
						_, err = w.StartBox(&mp4.BoxInfo{Type: mp4.BoxTypeData()})
						if err != nil {
							return err
						}

						var boxData mp4.Data
						switch v := val.(type) {
						case string:
							boxData.DataType = mp4.DataTypeStringUTF8
							boxData.Data = []byte(v)
						case uint8:
							boxData.DataType = mp4.DataTypeSignedIntBigEndian
							boxData.Data = []byte{v}
						case uint32:
							boxData.DataType = mp4.DataTypeSignedIntBigEndian
							boxData.Data = make([]byte, 4)
							binary.BigEndian.PutUint32(boxData.Data, v)
						case []byte:
							boxData.DataType = mp4.DataTypeBinary
							boxData.Data = v
						default:
							panic("unsupported value")
						}

						_, err = mp4.Marshal(w, &boxData, ctx)
						if err != nil {
							return err
						}

						_, err = w.EndBox()
						return err
					}

					addMeta := func(tag mp4.BoxType, val interface{}) error {
						_, err = w.StartBox(&mp4.BoxInfo{Type: tag})
						if err != nil {
							return err
						}

						err = marshalData(val)
						if err != nil {
							return err
						}

						_, err = w.EndBox()
						return err
					}

					addExtendedMeta := func(name string, val interface{}) error {
						ctx.UnderIlstFreeMeta = true

						_, err = w.StartBox(&mp4.BoxInfo{Type: mp4.BoxType{'-', '-', '-', '-'}, Context: ctx})
						if err != nil {
							return err
						}

						{
							_, err = w.StartBox(&mp4.BoxInfo{Type: mp4.BoxType{'m', 'e', 'a', 'n'}, Context: ctx})
							if err != nil {
								return err
							}

							_, err = w.Write([]byte{0, 0, 0, 0})
							if err != nil {
								return err
							}

							_, err = io.WriteString(w, "com.apple.iTunes")
							if err != nil {
								return err
							}

							_, err = w.EndBox()
							if err != nil {
								return err
							}
						}

						{
							_, err = w.StartBox(&mp4.BoxInfo{Type: mp4.BoxType{'n', 'a', 'm', 'e'}, Context: ctx})
							if err != nil {
								return err
							}

							_, err = w.Write([]byte{0, 0, 0, 0})
							if err != nil {
								return err
							}

							_, err = io.WriteString(w, name)
							if err != nil {
								return err
							}

							_, err = w.EndBox()
							if err != nil {
								return err
							}
						}

						err = marshalData(val)
						if err != nil {
							return err
						}

						ctx.UnderIlstFreeMeta = false

						_, err = w.EndBox()
						return err
					}

					err = addMeta(mp4.BoxType{'\251', 'n', 'a', 'm'}, meta.Attributes.Name)
					if err != nil {
						return err
					}

					err = addMeta(mp4.BoxType{'s', 'o', 'n', 'm'}, meta.Attributes.Name)
					if err != nil {
						return err
					}
					AlbumName := meta.Attributes.AlbumName
					//if strings.Contains(meta.ID, "pl.") {
					//	if !config.UseSongInfoForPlaylist {
					//		AlbumName = meta.Data[0].Attributes.Name
					//	}
					//}
					err = addMeta(mp4.BoxType{'\251', 'a', 'l', 'b'}, AlbumName)
					if err != nil {
						return err
					}

					err = addMeta(mp4.BoxType{'s', 'o', 'a', 'l'}, AlbumName)
					if err != nil {
						return err
					}

					err = addMeta(mp4.BoxType{'\251', 'A', 'R', 'T'}, meta.Attributes.ArtistName)
					if err != nil {
						return err
					}

					err = addMeta(mp4.BoxType{'s', 'o', 'a', 'r'}, meta.Attributes.ArtistName)
					if err != nil {
						return err
					}

					err = addMeta(mp4.BoxType{'\251', 'p', 'r', 'f'}, meta.Attributes.ArtistName)
					if err != nil {
						return err
					}

					err = addExtendedMeta("PERFORMER", meta.Attributes.ArtistName)
					if err != nil {
						return err
					}

					err = addExtendedMeta("ITUNESALBUMID", albums[0].ID)
					if err != nil {
						return err
					}

					err = addMeta(mp4.BoxType{'\251', 'w', 'r', 't'}, meta.Attributes.ComposerName)
					if err != nil {
						return err
					}

					err = addMeta(mp4.BoxType{'s', 'o', 'c', 'o'}, meta.Attributes.ComposerName)
					if err != nil {
						return err
					}

					err = addMeta(mp4.BoxType{'\251', 'd', 'a', 'y'}, meta.Attributes.ReleaseDate)
					if err != nil {
						return err
					}

					err = addExtendedMeta("RELEASETIME", meta.Attributes.ReleaseDate)
					if err != nil {
						return err
					}

					cnID, err := strconv.ParseUint(meta.ID, 10, 32)
					if err != nil {
						return err
					}

					err = addMeta(mp4.BoxType{'c', 'n', 'I', 'D'}, uint32(cnID))
					if err != nil {
						return err
					}

					err = addExtendedMeta("ISRC", meta.Attributes.ISRC)
					if err != nil {
						return err
					}

					if len(meta.Attributes.GenreNames) > 0 {
						err = addMeta(mp4.BoxType{'\251', 'g', 'e', 'n'}, meta.Attributes.GenreNames[0])
						if err != nil {
							return err
						}
					}

					if len(albums) > 0 {

						err = addMeta(mp4.BoxType{'a', 'A', 'R', 'T'}, meta.Attributes.ArtistName)
						if err != nil {
							return err
						}

						err = addMeta(mp4.BoxType{'s', 'o', 'a', 'a'}, meta.Attributes.ArtistName)
						if err != nil {
							return err
						}

						err = addMeta(mp4.BoxType{'c', 'p', 'r', 't'}, albums[0].Attributes.Copyright)
						if err != nil {
							return err
						}

						var isCpil uint8
						if albums[0].Attributes.IsCompilation {
							isCpil = 1
						}
						err = addMeta(mp4.BoxType{'c', 'p', 'i', 'l'}, isCpil)
						if err != nil {
							return err
						}

						err = addMeta(mp4.BoxType{'\251', 'p', 'u', 'b'}, albums[0].Attributes.RecordLabel)
						if err != nil {
							return err
						}

						err = addExtendedMeta("LABEL", albums[0].Attributes.RecordLabel)
						if err != nil {
							return err
						}

						err = addExtendedMeta("UPC", albums[0].Attributes.UPC)
						if err != nil {
							return err
						}

						//if !strings.Contains(meta.Data[0].ID, "pl.") {
						//	plID, err := strconv.ParseUint(meta.Data[0].ID, 10, 32)
						//	if err != nil {
						//		return err
						//	}
						//
						//	err = addMeta(mp4.BoxType{'p', 'l', 'I', 'D'}, uint32(plID))
						//	if err != nil {
						//		return err
						//	}
						//}
					}

					if len(artists) > 0 {
						if len(artists[0].ID) > 0 {
							atID, err := strconv.ParseUint(artists[0].ID, 10, 32)
							if err != nil {
								return err
							}

							err = addMeta(mp4.BoxType{'a', 't', 'I', 'D'}, uint32(atID))
							if err != nil {
								return err
							}
						}
					}
					trkn := make([]byte, 8)
					disk := make([]byte, 8)
					binary.BigEndian.PutUint32(trkn, uint32(meta.Attributes.TrackNumber))
					binary.BigEndian.PutUint16(trkn[4:], uint16(albums[0].Attributes.TrackCount))
					binary.BigEndian.PutUint32(disk, uint32(meta.Attributes.DiscNumber))
					//binary.BigEndian.PutUint16(disk[4:], uint16(meta.Data[0].Relationships.Tracks.Data[trackTotal-1].Attributes.DiscNumber))
					//if strings.Contains(meta.Data[0].ID, "pl.") {
					//	if !config.UseSongInfoForPlaylist {
					//		binary.BigEndian.PutUint32(trkn, uint32(trackNum))
					//		binary.BigEndian.PutUint16(trkn[4:], uint16(trackTotal))
					//		binary.BigEndian.PutUint32(disk, uint32(1))
					//		binary.BigEndian.PutUint16(disk[4:], uint16(1))
					//	}
					//}
					err = addMeta(mp4.BoxType{'t', 'r', 'k', 'n'}, trkn)
					if err != nil {
						return err
					}
					err = addMeta(mp4.BoxType{'d', 'i', 's', 'k'}, disk)
					if err != nil {
						return err
					}

					ctx.UnderIlst = false

					_, err = w.EndBox()
					if err != nil {
						return err
					}
				}

				ctx.UnderIlstMeta = false
				_, err = w.EndBox()
				if err != nil {
					return err
				}
			}

			ctx.UnderUdta = false
			_, err = w.EndBox()
			if err != nil {
				return err
			}
		}

		_, err = w.EndBox()
		if err != nil {
			return err
		}
	}

	{
		box, err := w.StartBox(&mp4.BoxInfo{Type: mp4.BoxTypeMdat()})
		if err != nil {
			return err
		}

		_, err = mp4.Marshal(w, &mp4.Mdat{Data: data}, box.Context)
		if err != nil {
			return err
		}

		mdat, err := w.EndBox()

		var realStco mp4.Stco

		offset := mdat.Offset + mdat.HeaderSize
		for i := uint32(0); i < numSamples; i++ {
			if i%chunkSize == 0 {
				realStco.EntryCount++
				realStco.ChunkOffset = append(realStco.ChunkOffset, uint32(offset))
			}
			offset += uint64(len(info.samples[i].data))
		}

		_, err = stco.SeekToPayload(w)
		if err != nil {
			return err
		}
		_, err = mp4.Marshal(w, &realStco, box.Context)
		if err != nil {
			return err
		}
	}

	return nil

}
