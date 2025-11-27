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
import no.nb.nna.veidemann.api.config.v1.Collection.SubCollection;
import no.nb.nna.veidemann.api.config.v1.Collection.SubCollectionType;
import no.nb.nna.veidemann.api.config.v1.ConfigObject;

import java.time.OffsetDateTime;
import java.time.ZoneOffset;
import java.time.ZonedDateTime;
import java.time.format.DateTimeFormatter;

public class CollectionNameGenerator {
    static final DateTimeFormatter HOUR_FORMAT = DateTimeFormatter.ofPattern("YYYYMMddHH");
    static final DateTimeFormatter DAY_FORMAT = DateTimeFormatter.ofPattern("YYYYMMdd");
    static final DateTimeFormatter MONTH_FORMAT = DateTimeFormatter.ofPattern("YYYYMM");
    static final DateTimeFormatter YEAR_FORMAT = DateTimeFormatter.ofPattern("YYYY");

    private CollectionNameGenerator() {
    }

    public static String getCollectionName(ConfigObject collectionConfig, SubCollectionType subType) {
        String collectionName = createBaseCollectionName(collectionConfig, OffsetDateTime.now(ZoneOffset.UTC));
        for (SubCollection sub : collectionConfig.getCollection().getSubCollectionsList()) {
            if (sub.getType() == subType) {
                return collectionName + "_" + sub.getName();
            }
        }
        return collectionName;
    }

    public static String createBaseCollectionName(ConfigObject collectionConfig, OffsetDateTime timestamp) {
        String name = collectionConfig.getMeta().getName();
        String dedupRotationKey = createCollectionRotationKey(collectionConfig.getCollection().getCollectionDedupPolicy(), timestamp);
        if (dedupRotationKey.isEmpty()) {
            return name;
        } else {
            return name + "_" + dedupRotationKey;
        }
    }

    public static String createCollectionRotationKey(RotationPolicy fileRotationPolicy, OffsetDateTime timestamp) {
        ZonedDateTime ts = timestamp.atZoneSameInstant(ZoneOffset.UTC);
        switch (fileRotationPolicy) {
            case HOURLY:
                return ts.format(HOUR_FORMAT);
            case DAILY:
                return ts.format(DAY_FORMAT);
            case MONTHLY:
                return ts.format(MONTH_FORMAT);
            case YEARLY:
                return ts.format(YEAR_FORMAT);
            default:
                return "";
        }
    }

}
