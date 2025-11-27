/*
 * Copyright 2017 National Library of Norway.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package no.nb.nna.veidemann.commons.util;

import no.nb.nna.veidemann.api.config.v1.Label;
import no.nb.nna.veidemann.api.config.v1.Meta;
import org.junit.jupiter.api.Test;

import static no.nb.nna.veidemann.commons.util.ApiTools.buildLabel;
import static no.nb.nna.veidemann.commons.util.ApiTools.buildMeta;
import static org.assertj.core.api.Assertions.assertThat;

/**
 *
 */
public class ApiToolsTest {
    @Test
    public void testGetFirstLabelWithKey() {
        Meta meta = buildMeta("name", "descr",
                buildLabel("aa", "bb"), buildLabel("cc", "dd"), buildLabel("aa", "ee"));

        assertThat(ApiTools.getFirstLabelWithKey(meta, "aa")).isPresent().contains(buildLabel("aa", "bb"));
        assertThat(ApiTools.getFirstLabelWithKey(meta, "cc")).isPresent().contains(buildLabel("cc", "dd"));
        assertThat(ApiTools.getFirstLabelWithKey(meta, "bb")).isNotPresent();
    }

    /**
     * Test of hasLabel method, of class ApiTools.
     */
    @Test
    public void testHasLabel() {
        Meta meta = buildMeta("name", "descr", buildLabel("aa", "bb"), buildLabel("cc", "dd"));

        Label labelToFind1 = buildLabel("aa", "bb");
        Label labelToFind2 = buildLabel("cc", "dd");
        Label labelToFind3 = buildLabel("ee", "ff");

        assertThat(ApiTools.hasLabel(meta, labelToFind1)).isTrue();

        assertThat(ApiTools.hasLabel(meta, labelToFind1, labelToFind2)).isTrue();

        assertThat(ApiTools.hasLabel(meta, labelToFind2, labelToFind1)).isTrue();

        assertThat(ApiTools.hasLabel(meta, labelToFind3)).isFalse();

        assertThat(ApiTools.hasLabel(meta, labelToFind3, labelToFind1)).isFalse();

        assertThat(ApiTools.hasLabel(meta)).isFalse();
    }
}