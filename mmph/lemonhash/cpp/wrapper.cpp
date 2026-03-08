#include "wrapper.h"

#include <vector>
#include <string>
#include <LeMonHashVL.hpp>

extern "C" {

LeMonHashPtr lemonhash_vl_new(const char* const* strings, const size_t* lengths, size_t num_strings) {
    std::vector<std::string> vec;
    vec.reserve(num_strings);
    for (size_t i = 0; i < num_strings; i++) {
        vec.emplace_back(strings[i], lengths[i]);
    }
    auto* instance = new lemonhash::LeMonHashVL<>(vec);
    return static_cast<LeMonHashPtr>(instance);
}

uint64_t lemonhash_vl_query(LeMonHashPtr ptr, const char* key_data, size_t key_len) {
    auto* instance = static_cast<lemonhash::LeMonHashVL<>*>(ptr);
    std::string key(key_data, key_len);
    return (*instance)(key);
}

size_t lemonhash_vl_space_bits(LeMonHashPtr ptr) {
    auto* instance = static_cast<lemonhash::LeMonHashVL<>*>(ptr);
    return instance->spaceBits();
}

void lemonhash_vl_free(LeMonHashPtr ptr) {
    auto* instance = static_cast<lemonhash::LeMonHashVL<>*>(ptr);
    delete instance;
}

}
