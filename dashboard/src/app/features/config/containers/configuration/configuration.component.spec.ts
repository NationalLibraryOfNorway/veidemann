import {Location} from '@angular/common';
import {ComponentFixture, TestBed} from '@angular/core/testing';
import {MatDialog} from '@angular/material/dialog';
import {ActivatedRoute, convertToParamMap, ParamMap, Router} from '@angular/router';
import {BehaviorSubject, EMPTY, of} from 'rxjs';

import {AuthService, ControllerApiService, ErrorService, SnackBarService} from '../../../../core';
import {ConfigObject, ConfigRef, Kind} from '../../../../shared/models';
import {ConfigService} from '../../../../shared/services';
import {OptionsService} from '../../services';
import {RouterExtraService} from '../../services/router-extra.service';
import {ConfigurationComponent} from './configuration.component';

describe('ConfigurationComponent route loading', () => {
  let fixture: ComponentFixture<ConfigurationComponent>;
  let idParams: BehaviorSubject<ParamMap>;
  let kindParams: BehaviorSubject<ParamMap>;

  const get = vi.fn((ref: ConfigRef) => of(new ConfigObject({id: ref.id, kind: ref.kind})));

  beforeEach(async () => {
    idParams = new BehaviorSubject(convertToParamMap({id: 'entity-1'}));
    kindParams = new BehaviorSubject(convertToParamMap({kind: 'entity'}));
    get.mockClear();

    await TestBed.configureTestingModule({
      imports: [ConfigurationComponent],
      providers: [
        {
          provide: ActivatedRoute,
          useValue: {paramMap: idParams, parent: {paramMap: kindParams}},
        },
        {provide: ConfigService, useValue: {get, loading$: of(false)}},
        {provide: OptionsService, useValue: {options$: of({})}},
        {provide: AuthService, useValue: {}},
        {provide: ControllerApiService, useValue: {}},
        {provide: ErrorService, useValue: {dispatch: vi.fn()}},
        {provide: SnackBarService, useValue: {openSnackBar: vi.fn()}},
        {provide: RouterExtraService, useValue: {}},
        {provide: Location, useValue: {}},
        {provide: Router, useValue: {events: EMPTY}},
        {provide: MatDialog, useValue: {}},
      ],
    })
      .overrideComponent(ConfigurationComponent, {set: {template: ''}})
      .compileComponents();

    fixture = TestBed.createComponent(ConfigurationComponent);
  });

  it('loads immediately and reloads when the selected route changes', async () => {
    const loaded: ConfigObject[] = [];
    fixture.componentInstance.configObject$.subscribe(config => loaded.push(config));
    fixture.detectChanges();
    await fixture.whenStable();

    expect(get).toHaveBeenLastCalledWith(expect.objectContaining({
      id: 'entity-1',
      kind: Kind.CRAWLENTITY,
    }));

    idParams.next(convertToParamMap({id: 'entity-2'}));
    fixture.detectChanges();
    await fixture.whenStable();

    expect(get).toHaveBeenCalledTimes(2);
    expect(get).toHaveBeenLastCalledWith(expect.objectContaining({
      id: 'entity-2',
      kind: Kind.CRAWLENTITY,
    }));
    expect(loaded.at(-1)).toEqual(expect.objectContaining({id: 'entity-2'}));
  });
});
