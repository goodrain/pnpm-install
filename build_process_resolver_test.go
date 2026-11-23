package pnpminstall_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	pnpminstall "github.com/goodrain/pnpm-install"
	"github.com/goodrain/pnpm-install/fakes"
	"github.com/paketo-buildpacks/packit/v2/scribe"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testBuildProcessResolver(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		workingDir string

		rebuild *fakes.BuildProcess
		install *fakes.BuildProcess
		ci      *fakes.BuildProcess

		resolver pnpminstall.BuildProcessResolver

		buffer *bytes.Buffer
	)

	it.Before(func() {
		var err error
		workingDir, err = os.MkdirTemp("", "working-dir")
		Expect(err).NotTo(HaveOccurred())

		buffer = bytes.NewBuffer(nil)
		logger := scribe.NewLogger(buffer)

		rebuild = &fakes.BuildProcess{}
		rebuild.ShouldRunCall.Returns.Sha = "rebuild-sha"

		install = &fakes.BuildProcess{}
		install.ShouldRunCall.Returns.Sha = "install-sha"

		ci = &fakes.BuildProcess{}
		ci.ShouldRunCall.Returns.Sha = "ci-sha"

		resolver = pnpminstall.NewBuildProcessResolver(logger, rebuild, install, ci)
	})

	it.After(func() {
		Expect(os.RemoveAll(workingDir)).To(Succeed())
	})

	context("the build process is install", func() {
		context("there is no node_modules, pnpm-lock.yaml, or cache", func() {
			it("returns the install process", func() {
				buildProcess, cacheUsed, err := resolver.Resolve(workingDir)
				Expect(err).NotTo(HaveOccurred())
				Expect(cacheUsed).To(BeFalse())

				Expect(buildProcess).To(Equal(install))

				Expect(buffer.String()).To(ContainSubstring("Selected PNPM build process: 'pnpm install'"))
			})
		})

		context("there is no node_modules, pnpm-lock.yaml but there is a cache", func() {
			it.Before(func() {
				Expect(os.MkdirAll(filepath.Join(workingDir, "pnpm-store"), os.ModePerm)).To(Succeed())
			})

			it("returns the install process", func() {
				buildProcess, cacheUsed, err := resolver.Resolve(workingDir)
				Expect(err).NotTo(HaveOccurred())
				Expect(cacheUsed).To(BeTrue())

				Expect(buildProcess).To(Equal(install))
			})
		})
	})

	context("the build process is rebuild", func() {
		context("there is no pnpm-lock.yaml or cache but there is node_modules", func() {
			it.Before(func() {
				Expect(os.MkdirAll(filepath.Join(workingDir, "node_modules"), os.ModePerm)).To(Succeed())
			})

			it("returns the rebuild process", func() {
				buildProcess, cacheUsed, err := resolver.Resolve(workingDir)
				Expect(err).NotTo(HaveOccurred())
				Expect(cacheUsed).To(BeFalse())

				Expect(buildProcess).To(Equal(rebuild))

				Expect(buffer.String()).To(ContainSubstring("Selected PNPM build process: 'pnpm rebuild'"))
			})
		})

		context("there is no pnpm-lock.yaml but both node_modules and a cache are present", func() {
			it.Before(func() {
				Expect(os.MkdirAll(filepath.Join(workingDir, "node_modules"), os.ModePerm)).To(Succeed())

				Expect(os.MkdirAll(filepath.Join(workingDir, "pnpm-store"), os.ModePerm)).To(Succeed())
			})

			it("returns the rebuild process", func() {
				buildProcess, cacheUsed, err := resolver.Resolve(workingDir)
				Expect(err).NotTo(HaveOccurred())
				Expect(cacheUsed).To(BeTrue())

				Expect(buildProcess).To(Equal(rebuild))
			})
		})

		context("there is no cache but node_modules and a pnpm-lock.yaml are present", func() {
			it.Before(func() {
				Expect(os.MkdirAll(filepath.Join(workingDir, "node_modules"), os.ModePerm)).To(Succeed())

				err := os.WriteFile(filepath.Join(workingDir, "pnpm-lock.yaml"), []byte("some-content"), 0644)
				Expect(err).NotTo(HaveOccurred())
			})

			it("returns the rebuild process", func() {
				buildProcess, cacheUsed, err := resolver.Resolve(workingDir)
				Expect(err).NotTo(HaveOccurred())
				Expect(cacheUsed).To(BeFalse())

				Expect(buildProcess).To(Equal(rebuild))
			})
		})
	})

	context("the build process is ci", func() {
		context("there is no node_modules or cache but there is a pnpm-lock.yaml", func() {
			it.Before(func() {
				err := os.WriteFile(filepath.Join(workingDir, "pnpm-lock.yaml"), []byte("some-content"), 0644)
				Expect(err).NotTo(HaveOccurred())
			})

			it("returns the ci process", func() {
				buildProcess, cacheUsed, err := resolver.Resolve(workingDir)
				Expect(err).NotTo(HaveOccurred())
				Expect(cacheUsed).To(BeFalse())

				Expect(buildProcess).To(Equal(ci))

				Expect(buffer.String()).To(ContainSubstring("Selected PNPM build process: 'pnpm install --frozen-lockfile'"))
			})
		})

		context("there is no node_modules but the pnpm-lock.yaml and cache are present", func() {
			it.Before(func() {
				err := os.WriteFile(filepath.Join(workingDir, "pnpm-lock.yaml"), []byte("some-content"), 0644)
				Expect(err).NotTo(HaveOccurred())

				Expect(os.MkdirAll(filepath.Join(workingDir, "pnpm-store"), os.ModePerm)).To(Succeed())
			})

			it("returns the ci process", func() {
				buildProcess, cacheUsed, err := resolver.Resolve(workingDir)
				Expect(err).NotTo(HaveOccurred())
				Expect(cacheUsed).To(BeTrue())

				Expect(buildProcess).To(Equal(ci))
			})
		})

		context("the pnpm-lock.yaml, node_modules, and cache are present", func() {
			it.Before(func() {
				Expect(os.MkdirAll(filepath.Join(workingDir, "node_modules"), os.ModePerm)).To(Succeed())

				err := os.WriteFile(filepath.Join(workingDir, "pnpm-lock.yaml"), []byte("some-content"), 0644)
				Expect(err).NotTo(HaveOccurred())

				Expect(os.MkdirAll(filepath.Join(workingDir, "pnpm-store"), os.ModePerm)).To(Succeed())
			})

			it("returns the ci process", func() {
				buildProcess, cacheUsed, err := resolver.Resolve(workingDir)
				Expect(err).NotTo(HaveOccurred())

				Expect(cacheUsed).To(BeTrue())
				Expect(buildProcess).To(Equal(ci))
			})
		})
	})

	context("output cases", func() {
		context("when there is a pnpm-lock.yaml", func() {
			it.Before(func() {
				err := os.WriteFile(filepath.Join(workingDir, "pnpm-lock.yaml"), []byte("some-content"), 0644)
				Expect(err).NotTo(HaveOccurred())
			})

			it("outputs that pnpm-lock.yaml was found", func() {
				_, _, err := resolver.Resolve(workingDir)
				Expect(err).NotTo(HaveOccurred())

				Expect(buffer.String()).To(MatchRegexp(`pnpm-lock.yaml\s+-> "Found"`))
			})
		})

		context("when there is a node_modules directory", func() {
			it.Before(func() {
				Expect(os.MkdirAll(filepath.Join(workingDir, "node_modules"), os.ModePerm)).To(Succeed())
			})

			it("outputs that pnpm-lock.yaml was found", func() {
				_, _, err := resolver.Resolve(workingDir)
				Expect(err).NotTo(HaveOccurred())

				Expect(buffer.String()).To(MatchRegexp(`node_modules\s+-> "Found"`))
			})
		})

		context("when there is a node_modules directory", func() {
			it.Before(func() {
				Expect(os.MkdirAll(filepath.Join(workingDir, "pnpm-store"), os.ModePerm)).To(Succeed())
			})

			it("outputs that pnpm-lock.yaml was found", func() {
				_, _, err := resolver.Resolve(workingDir)
				Expect(err).NotTo(HaveOccurred())

				Expect(buffer.String()).To(MatchRegexp(`pnpm-store\s+-> "Found"`))
			})
		})
	})

	context("failure cases", func() {
		var resolver pnpminstall.BuildProcessResolver

		it.Before(func() {
			var err error
			workingDir, err = os.MkdirTemp("", "working-dir")
			Expect(err).NotTo(HaveOccurred())

			logger := scribe.NewLogger(bytes.NewBuffer(nil))
			resolver = pnpminstall.NewBuildProcessResolver(logger, rebuild, install, ci)
		})

		it.After(func() {
			Expect(os.RemoveAll(workingDir)).To(Succeed())
		})

		context("Resolve", func() {
			context("when the working directory is unreadable", func() {
				it.Before(func() {
					Expect(os.Chmod(workingDir, 0000)).To(Succeed())
				})

				it.After(func() {
					Expect(os.Chmod(workingDir, os.ModePerm)).To(Succeed())
				})

				it("returns an error", func() {
					_, _, err := resolver.Resolve(workingDir)
					Expect(err).To(MatchError(ContainSubstring("permission denied")))
				})
			})
		})
	})
}
