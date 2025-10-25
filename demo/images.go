package main

import (
	"fmt"

	ops "github.com/patent-dev/epo-ops"
	"github.com/patent-dev/epo-ops/tiffutil"
)

// demoImages demonstrates Image Retrieval endpoints (2 endpoints)
func demoImages(demo *DemoContext) {
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("Image Retrieval Services (2 endpoints)")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	parts := ops.ParsePatentNumber(demo.ToEpodoc())
	if parts.Country == "" {
		fmt.Println("  ✗ Could not parse patent number")
		fmt.Println()
		return
	}

	// 1. GetImage - thumbnail (drawings, page 1)
	runEndpoint(demo, "get_image_thumbnail", "GetImage (thumbnail, page 1)",
		func() ([]byte, error) {
			return demo.Client.GetImage(demo.Ctx, parts.Country, parts.Number, parts.Kind, ops.ImageTypeThumbnail, 1)
		},
		FormatRequestDescription("GetImage", map[string]string{
			"country":   parts.Country,
			"number":    parts.Number,
			"kind":      parts.Kind,
			"imageType": ops.ImageTypeThumbnail,
			"page":      "1",
		}))

	// 2. GetImagePOST (POST method - keeps identifier encrypted in body)
	runEndpoint(demo, "get_image_post", "GetImagePOST",
		func() ([]byte, error) {
			// Format: CC/NNNNNNNN/KC/pagetype (e.g., EP/1000000/A1/fullimage)
			identifier := fmt.Sprintf("%s/%s/%s/fullimage", parts.Country, parts.Number, parts.Kind)
			return demo.Client.GetImagePOST(demo.Ctx, 1, identifier)
		},
		FormatRequestDescription("GetImagePOST", map[string]string{
			"page":       "1",
			"identifier": fmt.Sprintf("%s/%s/%s/fullimage", parts.Country, parts.Number, parts.Kind),
		}))

	// 3. Convert TIFF to PNG (if we have TIFF data)
	fmt.Printf("  → TIFF to PNG conversion... ")
	imageData, err := demo.Client.GetImage(demo.Ctx, parts.Country, parts.Number, parts.Kind, ops.ImageTypeThumbnail, 1)
	if err != nil {
		fmt.Printf("✗ Could not fetch image: %v\n", err)
	} else {
		format := DetectFormat(imageData)
		if format == FormatTIFF {
			pngData, err := tiffutil.TIFFToPNG(imageData)
			if err != nil {
				fmt.Printf("✗ Conversion failed: %v\n", err)
				demo.TotalCount++
				demo.FailureCount++
			} else {
				fmt.Printf("✓ %d bytes PNG\n", len(pngData))
				demo.TotalCount++
				demo.SuccessCount++

				// Save the PNG
				if !demo.SkipSave {
					requestDesc := FormatRequestDescription("TIFFToPNG", map[string]string{
						"input":  "TIFF from GetImage",
						"output": "PNG",
					})
					if err := demo.Saver.SaveExample("tiff_to_png", requestDesc, pngData, FormatPNG); err != nil {
						fmt.Printf("    Warning: Failed to save PNG: %v\n", err)
					}
				}
			}
		} else {
			fmt.Printf("✗ Image is %s, not TIFF (conversion skipped)\n", format)
			demo.TotalCount++
			demo.FailureCount++
		}
	}

	fmt.Println()
}
