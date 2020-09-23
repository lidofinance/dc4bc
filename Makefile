TMP_DIR?=/tmp/

OPENCV_VERSION?=4.4.0

PROJECT_DIR=$(pwd)

RPMS=cmake curl wget git gtk2-devel libpng-devel libjpeg-devel libtiff-devel tbb tbb-devel libdc1394-devel unzip
DEBS=unzip wget build-essential cmake curl git libgtk2.0-dev pkg-config libavcodec-dev libavformat-dev libswscale-dev libtbb2 libtbb-dev libjpeg-dev libpng-dev libtiff-dev libdc1394-22-dev

explain:
	@echo "For quick install with typical defaults of both OpenCV and GoCV, run 'make install'"

# Detect Linux distribution
distro_deps=
ifneq ($(shell which dnf 2>/dev/null),)
	distro_deps=deps_fedora
else
ifneq ($(shell which apt-get 2>/dev/null),)
	distro_deps=deps_debian
else
ifneq ($(shell which yum 2>/dev/null),)
	distro_deps=deps_rh_centos
endif
endif
endif

# Install all necessary dependencies.
deps: $(distro_deps)

deps_rh_centos:
	sudo yum -y install pkgconfig $(RPMS)

deps_fedora:
	sudo dnf -y install pkgconf-pkg-config $(RPMS)

deps_debian:
	sudo apt-get -y update
	sudo apt-get -y install $(DEBS)

download:
	sudo rm -rf $(TMP_DIR)opencv
	sudo mkdir $(TMP_DIR)opencv
	cd $(TMP_DIR)opencv
	curl -Lo opencv.zip https://github.com/opencv/opencv/archive/$(OPENCV_VERSION).zip
	unzip -q opencv.zip
	curl -Lo opencv_contrib.zip https://github.com/opencv/opencv_contrib/archive/$(OPENCV_VERSION).zip
	unzip -q opencv_contrib.zip
	rm opencv.zip opencv_contrib.zip
	cd -

test:
	@echo "Testing Go packages..."
	@go test ./... -cover

test-short:
	@echo "Testing Go packages..."
	@go test ./app/... -cover -short

mocks:
	@echo "Regenerate mocks..."
	@go generate ./...

build-darwin:
	@echo "Building dc4bc_d..."
	GOOS=darwin GOARCH=amd64 go build -o dc4bc_d_darwin ./cmd/dc4bc_d/main.go
	@echo "Building dc4bc_cli..."
	GOOS=darwin GOARCH=amd64 go build -o dc4bc_cli_darwin ./cmd/dc4bc_cli/main.go
	@echo "Building dc4bc_airgapped..."
	GOOS=darwin GOARCH=amd64 go build -o dc4bc_airgapped_darwin ./cmd/airgapped/main.go

build-linux:
	@echo "Building dc4bc_d..."
	GOOS=linux GOARCH=amd64 go build -o dc4bc_d_linux ./cmd/dc4bc_d/main.go
	@echo "Building dc4bc_cli..."
	GOOS=linux GOARCH=amd64 go build -o dc4bc_cli_linux ./cmd/dc4bc_cli/main.go
	@echo "Building dc4bc_airgapped..."
	GOOS=linux GOARCH=amd64 go build -o dc4bc_airgapped_linux ./cmd/airgapped/main.go

clean:
	go clean --cache
	rm -rf $(TMP_DIR)opencv

sudo_pre_install_clean:
	sudo rm -rf /usr/local/lib/cmake/opencv4/
	sudo rm -rf /usr/local/lib/libopencv*
	sudo rm -rf /usr/local/lib/pkgconfig/opencv*
	sudo rm -rf /usr/local/include/opencv*

build-linux-static: deps sudo_pre_install_clean download
	cd $(TMP_DIR)opencv/opencv-$(OPENCV_VERSION)
	mkdir build
	cd build
	rm -rf *
	cmake -D CMAKE_BUILD_TYPE=RELEASE -D CMAKE_INSTALL_PREFIX=/usr/local -D BUILD_SHARED_LIBS=OFF -D OPENCV_EXTRA_MODULES_PATH=$(TMP_DIR)opencv/opencv_contrib-$(OPENCV_VERSION)/modules -D BUILD_DOCS=OFF -D BUILD_EXAMPLES=OFF -D BUILD_TESTS=OFF -D BUILD_PERF_TESTS=OFF -D BUILD_opencv_java=NO -D WITH_FFMPEG=OFF -D WITH_QT=OFF -D WITH_GTK=OFF -D WITH_CUDA=OFF -D WITH_TIFF=OFF -D WITH_WEBP=OFF -D WITH_QT=OFF -D WITH_PNG=OFF -D WITH_1394=OFF -D HAVE_OPENEXR=OFF -D BUILD_opencv_python=NO -D BUILD_opencv_python2=NO -D BUILD_opencv_python3=NO -D WITH_JASPER=OFF -D OPENCV_GENERATE_PKGCONFIG=YES ..
	$(MAKE) -j $(shell nproc --all)
	$(MAKE) preinstall

	cd ${PROJECT_DIR}

	export CGO_CPPFLAGS="-I/usr/local/include/opencv4"
	export CGO_LDFLAGS="-L/usr/local/lib -L/usr/local/lib/opencv4/3rdparty -L/tmp/opencv/opencv-4.4.0/build/lib -lopencv_gapi -lopencv_stitching -lopencv_aruco -lopencv_bgsegm -lopencv_bioinspired -lopencv_ccalib -lopencv_dnn_objdetect -lopencv_dnn_superres -lopencv_dpm -lopencv_highgui -lopencv_face -lopencv_freetype -lopencv_fuzzy -lopencv_hfs -lopencv_img_hash -lopencv_intensity_transform -lopencv_line_descriptor -lopencv_quality -lopencv_rapid -lopencv_reg -lopencv_rgbd -lopencv_saliency -lopencv_stereo -lopencv_structured_light -lopencv_phase_unwrapping -lopencv_superres -lopencv_optflow -lopencv_surface_matching -lopencv_tracking -lopencv_datasets -lopencv_text -lopencv_dnn -lopencv_plot -lopencv_videostab -lopencv_videoio -lopencv_xfeatures2d -lopencv_shape -lopencv_ml -lopencv_ximgproc -lopencv_video -lopencv_xobjdetect -lopencv_objdetect -lopencv_calib3d -lopencv_imgcodecs -lopencv_features2d -lopencv_flann -lopencv_xphoto -lopencv_photo -lopencv_imgproc -lopencv_core -littnotify -llibprotobuf -lIlmImf -lquirc -lippiw -lippicv -lade -lgtk-x11-2.0 -lgdk-x11-2.0 -lpangocairo-1.0 -lcairo -lgio-2.0 -lpangoft2-1.0 -lpango-1.0 -lgobject-2.0 -lglib-2.0 -lfontconfig -lgthread-2.0 -lz -ljpeg -lfreetype -lharfbuzz -ldl -lm -lpthread -lrt"
	@echo "Building dc4bc_d..."
	go build -ldflags "-linkmode 'external' -extldflags '-static'" -o dc4bc_d_linux ./cmd/dc4bc_d/main.go
	@echo "Building dc4bc_cli..."
	go build -ldflags "-linkmode 'external' -extldflags '-static'" -o dc4bc_cli_linux ./cmd/dc4bc_cli/main.go
	@echo "Building dc4bc_airgapped..."
	go build -ldflags "-linkmode 'external' -extldflags '-static'" -o dc4bc_airgapped_linux ./cmd/airgapped/main.go


.PHONY: mocks
