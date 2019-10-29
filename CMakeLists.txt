cmake_minimum_required(VERSION 3.8)
project(Gunkan_ObjectStorage CXX)

set(CMAKE_CXX_STANDARD 17)

include(./Helpers.cmake)
include(CheckIncludeFileCXX)
find_package(PkgConfig)
find_package (Threads)

# JFS: glog is distributed the right way, let's rely on pkg-config
pkg_check_modules(GLOG libglog)
pkg_check_modules(GFLAGS libgflags)
pkg_check_modules(GFLAGS libgflags)

# JFS: Nearly no dependency brought by the system, that is the price to deal
#      with tools in too early stages
option_or_system(GFLAGS gflags gflags/gflags.h)
option_or_system(GLOG glog glog/logging.h)
option_or_system(NNG nng nng/nng.h)
option_or_system(FLATBUFFERS flatbuffers flatbuffers/flatbuffers.h)
option_or_system(ROCKSDB rocksdb rocksdb/db.h)
option_or_system(DILL dill libdill.h)
option_or_system(OPENTRACING opentracing opentracing/tracer.h)
option_or_system(OPENTRACING_MOCK opentracing_mocktracer opentracing/mocktracer/tracer.h)
option_or_system(HTTP http_parser http_parser.h)

check_found("GFLAGS" "GLOG" "NNG" "FLATBUFFERS" "ROCKSDB" "DILL")

check_include_file_cxx(flatbuffers/flatbuffers.h FLATC_HEADER)

find_program(FLATC_EXE flatc)
find_program(GOLANG_EXE go)

ENABLE_TESTING()

add_definitions(-D_XOPEN_SOURCE=700)

set(CMAKE_CXX_FLAGS "${CMAKE_CXX_FLAGS} -g -pipe -Wall -Wextra -Werror")
set(CMAKE_CXX_FLAGS "${CMAKE_CXX_FLAGS} -fPIC -fPIE -pie")
set(CMAKE_CXX_FLAGS "${CMAKE_CXX_FLAGS} -Wno-unused-variable")

set(CMAKE_CXX_FLAGS_DEBUG          "-O0 -fno-inline")
set(CMAKE_CXX_FLAGS_RELWITHDEBINFO "-O2 -fno-inline")
set(CMAKE_CXX_FLAGS_RELEASE        "-O2")
set(CMAKE_CXX_FLAGS_MINSIZEREL     "-Os -s")

add_definitions(-D_GNU_SOURCE)
add_definitions(-DDILL_DISABLE_RAW_NAMES)

include_directories(AFTER
    ${NNG_INCLUDE_DIRS}
    ${ROCKSDB_INCLUDE_DIRS}
    ${FLATBUFFERS_INCLUDE_DIRS}
    ${GLOG_INCLUDE_DIRS}
    ${GFLAGS_INCLUDE_DIRS})

include_directories(BEFORE
    ${CMAKE_SOURCE_DIR})

link_directories(
    ${NNG_LIBRARY_DIRS}
    ${ROCKSDB_LIBRARY_DIRS}
    ${FLATBUFFERS_LIBRARY_DIRS}
    ${GLOG_LIBRARY_DIRS} ${GFLAGS_LIBRARY_DIRS})

# KV ------------------------------------------------------------------------

add_custom_command(
    OUTPUT ${CMAKE_SOURCE_DIR}/kv/server/protocol_generated.h
    DEPENDS ${CMAKE_SOURCE_DIR}/kv/protocol.fbs
    COMMENT "Generating FlatBuffers codec for C++"
    COMMAND ${FLATC_EXE} --cpp
    ARGS
      -o ${CMAKE_SOURCE_DIR}/kv/server
      ${CMAKE_SOURCE_DIR}/kv/protocol.fbs)

add_custom_command(
    OUTPUT ${CMAKE_SOURCE_DIR}/kv/client/golang/protocol_generated.go
    DEPENDS ${CMAKE_SOURCE_DIR}/kv/protocol.fbs
    COMMENT "Generating FlatBuffers codec for Golang"
    COMMAND ${FLATC_EXE} --go
    ARGS
      --go-namespace kv --gen-onefile
      -o ${CMAKE_SOURCE_DIR}/kv/client/golang
      ${CMAKE_SOURCE_DIR}/kv/protocol.fbs)

add_custom_target(gunkan-kv-client ALL
    DEPENDS ${CMAKE_SOURCE_DIR}/kv/client/golang/protocol_generated.go
    WORKING_DIRECTORY ${CMAKE_SOURCE_DIR}/kv/cli
    COMMAND ${GOLANG_EXE} build -o ${CMAKE_BINARY_DIR}/gunkan-kv-client)

add_executable(gunkan-kv
    kv/server/main.cpp
    ${CMAKE_SOURCE_DIR}/kv/server/protocol_generated.h)

target_link_libraries(gunkan-kv
    -static-libgcc
    -static-libstdc++
    ${NNG_LIBRARIES}
    ${ROCKSDB_LIBRARIES}
    ${FLATBUFFERS_LIBRARIES}
    ${GLOG_LIBRARIES} ${GFLAGS_LIBRARIES}
    ${CMAKE_THREAD_LIBS_INIT})

install(TARGETS gunkan-kv RUNTIME DESTINATION bin)
install(SCRIPT gunkan-kv-client RUNTIME DESTINATION bin)

add_test(NAME kv/client/api
		COMMAND go test
		WORKING_DIRECTORY ${CMAKE_SOURCE_DIR}/kv/client/golang)


# BLOB ----------------------------------------------------------------------

add_executable(gunkan-blob
    blob/server/time.cpp
    blob/server/strings.cpp
    blob/server/charset.cpp
    blob/server/bytes.cpp
    blob/server/io.cpp
    blob/server/blobid.cpp
    blob/server/http.cpp
    blob/server/blob.cpp
    blob/server/thread_executor.cpp
    blob/server/thread_classifier.cpp
    blob/server/threads.cpp
    blob/server/main.cpp)

target_link_libraries(gunkan-blob
    -static-libgcc
    -static-libstdc++
    ${HTTP_LIBRARIES}
    ${GLOG_LIBRARIES} ${GFLAGS_LIBRARIES}
    ${DILL_LIBRARIES}
    ${OPENTRACING_LIBRARIES}
    ${OPENTRACING_MOCK_LIBRARIES}
    ${CMAKE_THREAD_LIBS_INIT})

add_custom_target(gunkan-blob-client ALL
		WORKING_DIRECTORY ${CMAKE_SOURCE_DIR}/blob/cli
		COMMAND ${GOLANG_EXE} build -o ${CMAKE_BINARY_DIR}/gunkan-blob-client)

install(TARGETS gunkan-blob RUNTIME DESTINATION bin)
install(SCRIPT gunkan-blob-client RUNTIME DESTINATION bin)

add_test(NAME blob/client/api
         COMMAND go test
         WORKING_DIRECTORY ${CMAKE_SOURCE_DIR}/blob/client/golang)
