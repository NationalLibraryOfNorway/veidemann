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

package no.nb.nna.veidemann.db;

import com.rethinkdb.RethinkDB;
import com.rethinkdb.net.Result;
import no.nb.nna.veidemann.api.config.v1.ConfigObject;
import no.nb.nna.veidemann.commons.db.ChangeFeed;
import org.junit.jupiter.api.Test;

import java.util.Map;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.TimeoutException;
import java.util.function.Function;

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.anyLong;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.when;

public class ChangeFeedBaseTest {
    static final RethinkDB r = RethinkDB.r;

    @Test
    public void stream() throws TimeoutException {
        Result<Map<String, Object>> cursorMock = mock(Result.class);
        when(cursorMock.hasNext())
                .thenReturn(true)
                .thenReturn(true)
                .thenReturn(true)
                .thenReturn(true)
                .thenReturn(false);
        when(cursorMock.next(anyLong(), any(TimeUnit.class)))
            .thenReturn(rethinkMap(r.hashMap("id", "id1")))
            .thenReturn(rethinkMap(r.hashMap("id", "id2").with("seed", r.hashMap("disabled", true))))
                // Error expected since boolean field disabled cannot contain a number
            .thenReturn(rethinkMap(r.hashMap("id", "id3").with("seed", r.hashMap("disabled", "100"))))
            .thenReturn(rethinkMap(r.hashMap("id", "id4").with("seed", r.hashMap("disabled", false))));

        ChangeFeed<ConfigObject> cf = new ChangeFeedBase<ConfigObject>(RethinkDbResultSet.fromResult(cursorMock)) {
            @Override
            protected Function<Map<String, Object>, ConfigObject> mapper() {
                return co -> {
                    try {
                        ConfigObject res = ProtoUtils.rethinkToProto(co, ConfigObject.class);
                        return res;
                    } catch (Exception e) {
                        // Expecting conversion of 'id3' to fail
                        assertThat(co.get("id")).isEqualTo("id3");
                        return null;
                    }
                };
            }
        };

        // Expecting only three because 'id3' should fail
        assertThat(cf.stream()).hasSize(3).extracting("id")
                .satisfies(objects -> {
                    assertThat(objects.get(0)).isEqualTo("id1");
                    assertThat(objects.get(1)).isEqualTo("id2");
                    assertThat(objects.get(2)).isEqualTo("id4");
                });
    }

    @SuppressWarnings("unchecked")
    private static Map<String, Object> rethinkMap(Object value) {
        return (Map<String, Object>) value;
    }
}