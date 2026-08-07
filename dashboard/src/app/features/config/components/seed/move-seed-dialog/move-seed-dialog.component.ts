import {AsyncPipe} from '@angular/common';
import {ChangeDetectionStrategy, Component, inject, signal} from '@angular/core';
import {FormControl, ReactiveFormsModule} from '@angular/forms';
import {MatButtonModule} from '@angular/material/button';
import {MAT_DIALOG_DATA, MatDialogModule} from '@angular/material/dialog';
import {MatFormFieldModule} from '@angular/material/form-field';
import {MatIcon} from '@angular/material/icon';
import {MatInput} from '@angular/material/input';
import {MatListModule, MatSelectionListChange} from '@angular/material/list';
import {MatProgressBar} from '@angular/material/progress-bar';
import {defer, Observable} from 'rxjs';
import {debounceTime, distinctUntilChanged, filter, finalize, map, shareReplay, startWith, switchMap, tap, toArray} from 'rxjs/operators';

import {ConfigQuery} from '../../../../../shared/func';
import {ConfigObject, ConfigRef, Kind} from '../../../../../shared/models';
import {ConfigService} from '../../../../../shared/services';

export interface MoveSeedDialogData {
  seed: ConfigObject;
}

export type MoveSeedDialogResult = ConfigRef;

@Component({
  selector: 'app-move-seed-dialog',
  templateUrl: './move-seed-dialog.component.html',
  styleUrls: ['./move-seed-dialog.component.scss'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    AsyncPipe,
    MatButtonModule,
    MatDialogModule,
    MatFormFieldModule,
    MatIcon,
    MatInput,
    MatListModule,
    MatProgressBar,
    ReactiveFormsModule,
  ],
  standalone: true,
})
export class MoveSeedDialogComponent {
  private readonly configService = inject(ConfigService);
  readonly data = inject<MoveSeedDialogData>(MAT_DIALOG_DATA);

  readonly search = new FormControl('', {nonNullable: true});
  readonly loading = signal(false);
  selectedEntity: ConfigObject | null = null;

  readonly entities$: Observable<ConfigObject[]> = this.search.valueChanges.pipe(
    startWith(''),
    tap(() => this.selectedEntity = null),
    map(term => term.trim()),
    debounceTime(250),
    distinctUntilChanged(),
    switchMap(term => defer(() => {
      this.loading.set(true);
      return this.configService.search(this.entityQuery(term), {offset: 0, pageSize: 50}).pipe(
        filter((entity): entity is ConfigObject => !!entity),
        toArray(),
        map(entities => entities.filter(entity => entity.id !== this.data.seed.seed.entityRef.id)),
        finalize(() => this.loading.set(false)),
      );
    })),
    shareReplay({bufferSize: 1, refCount: true}),
  );

  onSelectionChange(event: MatSelectionListChange): void {
    this.selectedEntity = event.options[0]?.value ?? null;
  }

  clearSearch(): void {
    this.search.setValue('');
  }

  result(): MoveSeedDialogResult | null {
    return this.selectedEntity ? ConfigObject.toConfigRef(this.selectedEntity) : null;
  }

  private entityQuery(term: string): ConfigQuery {
    return {
      kind: Kind.CRAWLENTITY,
      entityId: null,
      scheduleId: null,
      crawlConfigId: null,
      collectionId: null,
      browserConfigId: null,
      politenessId: null,
      disabled: null,
      browserScriptType: null,
      crawlJobIdList: [],
      scriptIdList: [],
      term: term || null,
      active: 'name',
      direction: 'asc',
    };
  }
}
