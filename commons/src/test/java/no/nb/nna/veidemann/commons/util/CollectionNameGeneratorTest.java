/*
 * Copyright 2019 National Library of Norway.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *       http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package no.nb.nna.veidemann.commons.util;

import no.nb.nna.veidemann.api.config.v1.Collection.RotationPolicy;
import no.nb.nna.veidemann.api.config.v1.Collection.SubCollectionType;
import no.nb.nna.veidemann.api.config.v1.ConfigObject;
import org.junit.jupiter.api.Test;

import java.time.OffsetDateTime;

import static org.assertj.core.api.Assertions.assertThat;

public class CollectionNameGeneratorTest {

    @Test
    public void getCollectionName() {
        String thisYear = String.valueOf(OffsetDateTime.now().getYear());
        ConfigObject.Builder conf = ConfigObject.newBuilder();
        conf.getMetaBuilder().setName("Test");

        ConfigObject config = conf.build();
        assertThat(CollectionNameGenerator.getCollectionName(config, SubCollectionType.UNDEFINED)).isEqualTo("Test");
        assertThat(CollectionNameGenerator.getCollectionName(config, SubCollectionType.DNS)).isEqualTo("Test");
        assertThat(CollectionNameGenerator.getCollectionName(config, SubCollectionType.SCREENSHOT)).isEqualTo("Test");

        conf.getCollectionBuilder().setCollectionDedupPolicy(RotationPolicy.YEARLY);
        config = conf.build();
        assertThat(CollectionNameGenerator.getCollectionName(config, SubCollectionType.UNDEFINED)).isEqualTo("Test_" + thisYear);
        assertThat(CollectionNameGenerator.getCollectionName(config, SubCollectionType.DNS)).isEqualTo("Test_" + thisYear);
        assertThat(CollectionNameGenerator.getCollectionName(config, SubCollectionType.SCREENSHOT)).isEqualTo("Test_" + thisYear);

        conf.getCollectionBuilder().addSubCollectionsBuilder().setName("sc").setType(SubCollectionType.SCREENSHOT);
        config = conf.build();
        assertThat(CollectionNameGenerator.getCollectionName(config, SubCollectionType.UNDEFINED)).isEqualTo("Test_" + thisYear);
        assertThat(CollectionNameGenerator.getCollectionName(config, SubCollectionType.DNS)).isEqualTo("Test_" + thisYear);
        assertThat(CollectionNameGenerator.getCollectionName(config, SubCollectionType.SCREENSHOT)).isEqualTo("Test_" + thisYear + "_sc");

        conf.getCollectionBuilder().addSubCollectionsBuilder().setName("dns").setType(SubCollectionType.DNS);
        config = conf.build();
        assertThat(CollectionNameGenerator.getCollectionName(config, SubCollectionType.UNDEFINED)).isEqualTo("Test_" + thisYear);
        assertThat(CollectionNameGenerator.getCollectionName(config, SubCollectionType.DNS)).isEqualTo("Test_" + thisYear + "_dns");
        assertThat(CollectionNameGenerator.getCollectionName(config, SubCollectionType.SCREENSHOT)).isEqualTo("Test_" + thisYear + "_sc");
    }

    @Test
    public void createBaseCollectionName() {
        ConfigObject.Builder conf = ConfigObject.newBuilder();
        conf.getMetaBuilder().setName("Test");
        conf.getCollectionBuilder().setCollectionDedupPolicy(RotationPolicy.DAILY);

        OffsetDateTime date = OffsetDateTime.parse("2018-06-18T14:38:16.667+02:00");

        assertThat(CollectionNameGenerator.createBaseCollectionName(conf.build(), date)).isEqualTo("Test_20180618");
    }

    @Test
    public void createFileRotationKey() {
        OffsetDateTime date = OffsetDateTime.parse("2018-06-18T14:38:16.667+02:00");

        assertThat(CollectionNameGenerator.createCollectionRotationKey(RotationPolicy.NONE, date)).isEqualTo("");
        assertThat(CollectionNameGenerator.createCollectionRotationKey(RotationPolicy.YEARLY, date)).isEqualTo("2018");
        assertThat(CollectionNameGenerator.createCollectionRotationKey(RotationPolicy.MONTHLY, date)).isEqualTo("201806");
        assertThat(CollectionNameGenerator.createCollectionRotationKey(RotationPolicy.DAILY, date)).isEqualTo("20180618");

        assertThat(CollectionNameGenerator.createCollectionRotationKey(RotationPolicy.HOURLY, date)).isEqualTo("2018061812");
        date = OffsetDateTime.parse("2018-06-18T12:38:16.667Z");
        assertThat(CollectionNameGenerator.createCollectionRotationKey(RotationPolicy.HOURLY, date)).isEqualTo("2018061812");
    }
}